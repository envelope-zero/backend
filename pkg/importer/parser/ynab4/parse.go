package ynab4

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	internal_types "github.com/envelope-zero/backend/v2/internal/types"
	"github.com/google/uuid"

	"github.com/envelope-zero/backend/v2/pkg/importer/types"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/text/currency"
)

// This function parses the comdirect CSV files.
func Parse(f io.Reader) (types.ParsedResources, error) {
	content, err := io.ReadAll(f)
	if err != nil {
		return types.ParsedResources{}, fmt.Errorf("could not read data from file: %w", err)
	}

	var budget Budget
	err = json.Unmarshal(content, &budget)
	if err != nil {
		return types.ParsedResources{}, fmt.Errorf("not a valid YNAB4 Budget.yfull file: %w", err)
	}

	var resources types.ParsedResources

	// Set options for the budget
	cur, _ := currency.FromTag(budget.BudgetMetaData.CurrencyLocale)
	resources.Budget = models.Budget{
		BudgetCreate: models.BudgetCreate{
			Currency: fmt.Sprintf("%s", currency.Symbol(cur)),
		},
	}

	// Parse accounts and payees
	accountIDNames := parseAccounts(&resources, budget.Accounts)
	payeeIDNames := parsePayees(&resources, budget.Payees)

	// Copy all payee mappings to the account mappings as for Envelope Zero, both are accounts
	maps.Copy(accountIDNames, payeeIDNames)

	envelopeIDNames, err := parseCategories(&resources, budget.Categories)
	if err != nil {
		return types.ParsedResources{}, fmt.Errorf("error parsing categories and subcategories: %w", err)
	}

	err = parseTransactions(&resources, budget.Transactions, accountIDNames, envelopeIDNames)
	if err != nil {
		return types.ParsedResources{}, fmt.Errorf("error parsing transactions: %w", err)
	}

	err = parseMonthlyBudgets(&resources, budget.MonthlyBudgets, envelopeIDNames)
	if err != nil {
		return types.ParsedResources{}, fmt.Errorf("error parsing budget allocations: %w", err)
	}

	// Translate YNAB overspend handling behaviour to EZ overspending handling behaviour
	fixOverspendHandling(&resources)

	return resources, nil
}

func parseHiddenCategoryName(f string) (category, envelope string, err error) {
	// The format of hidden category strings is shown in the next line. Square brackets denote field names
	// [Master Category Name] ` [Category Name] ` [Archival Number]
	match := regexp.MustCompile("(.*) ` (.*) `").FindStringSubmatch(f)

	// len needs to be 3 as the whole regex match is in match[0]
	if len(match) != 3 {
		return "", "", fmt.Errorf("incorrect hidden category format: match length is %d", len(match))
	}

	category = match[1]
	envelope = match[2]
	return
}

func parseAccounts(resources *types.ParsedResources, accounts []Account) IDToName {
	idToNames := make(IDToName)

	for _, account := range accounts {
		idToNames[account.EntityID] = account.Name

		resources.Accounts = append(resources.Accounts, models.Account{
			AccountCreate: models.AccountCreate{
				Name:       account.Name,
				Note:       account.Note,
				OnBudget:   account.OnBudget,
				Hidden:     account.Hidden,
				ImportHash: fmt.Sprint(sha256.Sum256([]byte(account.EntityID))),
			},
		})
	}

	return idToNames
}

func parsePayees(resources *types.ParsedResources, payees []Payee) IDToName {
	idToNames := make(IDToName)

	// Payees in YNAB 4 map to External Accounts in Envelope Zero
	for _, payee := range payees {
		idToNames[payee.EntityID] = payee.Name

		// Transfers are also stored as Payees with an entity ID of "Payee/Transfer:[Target account ID]"
		// As we do not need this hack for Envelope Zero, we skip those Payees
		//
		// We also do not need a magic "Starting Balance" payee since this is a feature of accounts
		if payee.Name == "Starting Balance" || strings.HasPrefix(payee.EntityID, "Payee/Transfer") {
			continue
		}

		// Create the account
		resources.Accounts = append(resources.Accounts, models.Account{
			AccountCreate: models.AccountCreate{
				Name:       payee.Name,
				OnBudget:   false,
				External:   true,
				ImportHash: fmt.Sprint(sha256.Sum256([]byte(payee.EntityID))),
			},
		})
	}

	return idToNames
}

func parseCategories(resources *types.ParsedResources, categories []Category) (IDToEnvelopes, error) {
	idToEnvelope := make(IDToEnvelopes)

	// Create temporary variables to hold all the parsed
	// data. They will be added to the ParsedResources
	// when parsing is complete.
	tCategories := make(map[string]types.Category)
	type tEnvelope struct {
		Envelope types.Envelope
		Category string
	}
	var tEnvelopes []tEnvelope

	for _, category := range categories {
		// The name "Pre-YNAB Debt" is used for a category created by YNAB for the starting balances
		// of accounts that have a negative starting balance. Since accounts on Enevelope Zero have
		// a starting balance that is not a transaction with a "magic" payee, this category is
		// not needed.
		if category.Name == "Pre-YNAB Debt" {
			continue
		}

		// Add the category
		tCategories[category.Name] = types.Category{
			Model: models.Category{
				CategoryCreate: models.CategoryCreate{
					Name: category.Name,
					Note: category.Note,
					// we use category.Deleted here since the original data format does not have a hidden field. If the category is not referenced anywhere,
					// it will not be imported anyway
					Hidden: category.Deleted,
				},
			},
			Envelopes: make(map[string]types.Envelope),
		}

		// Add the envelopes
		for _, envelope := range category.SubCategories {
			if envelope.Deleted {
				continue
			}

			// Map the envelope ID to the category and envelope name
			mapping := IDToEnvelope{
				Envelope: envelope.Name,
				Category: category.Name,
			}

			// For hidden categories, we need to extract the actual name
			var hidden bool
			if category.Name == "Hidden Categories" {
				var err error
				mapping.Category, mapping.Envelope, err = parseHiddenCategoryName(mapping.Envelope)
				if err != nil {
					return IDToEnvelopes{}, fmt.Errorf("hidden category could not be parsed, your Budget.yfull file seems to be corrupted: %w", err)
				}

				hidden = true
			}

			idToEnvelope[envelope.EntityID] = mapping

			tEnvelopes = append(tEnvelopes, tEnvelope{
				types.Envelope{
					Model: models.Envelope{
						EnvelopeCreate: models.EnvelopeCreate{
							Name:   mapping.Envelope,
							Note:   envelope.Note,
							Hidden: hidden,
						},
					},
				},
				mapping.Category,
			})
		}
	}

	// Initialize the categories
	resources.Categories = make(map[string]types.Category)

	// Add all envelopes, adding categories as needed
	for _, envelope := range tEnvelopes {
		// Check if the category already exists in the resources. If not, create it
		_, ok := resources.Categories[envelope.Category]
		if !ok {
			resources.Categories[envelope.Category] = tCategories[envelope.Category]
		}

		resources.Categories[envelope.Category].Envelopes[envelope.Envelope.Model.Name] = envelope.Envelope
	}

	return idToEnvelope, nil
}

func parseTransactions(resources *types.ParsedResources, transactions []Transaction, accountIDNames IDToName, envelopeIDNames IDToEnvelopes) error {
	// If an account "No payee" for transactions without a payee needs to be added
	addNoPayee := false

	// We generate an import hash for the "YNAB 4 - No payee" account that might be added since
	// we create it and it therefore does not have a UUID
	noPayeeImportHash := fmt.Sprint(sha256.Sum256([]byte(uuid.New().String())))

	// Add all transactions
	for _, transaction := range transactions {
		// Don't import deleted transactions or transactions that have an amount of 0
		//
		// Transfers create two corresponding transaction entries in YNAB 4
		//
		// They use the same entityId, but one is suffixed with "_T_0"
		// Therefore, we ignore transactions where the entity ID ends in
		// "_T_0"
		if transaction.Deleted || transaction.Amount.IsZero() || strings.HasSuffix(transaction.EntityID, "_T_0") {
			continue
		}

		// For transfers, the payee string has the prefix "Payee/Transfer:",
		// the actual account is stored in the TargetAccountID
		if strings.HasPrefix(transaction.PayeeID, "Payee/Transfer:") {
			transaction.PayeeID = transaction.TargetAccountID
		}

		// Initialize to the import hash of the "No payee"
		// account since we already have that one
		payeeImportHash := noPayeeImportHash

		// If the PayeeID is actually empty, create the account later
		if transaction.PayeeID == "" {
			addNoPayee = true
		} else {
			// Use the payee ID from the transaction in all other cases
			payeeImportHash = fmt.Sprint(sha256.Sum256([]byte(transaction.PayeeID)))
		}

		// Parse the date of the transaction
		date, err := time.Parse("2006-01-02", transaction.Date)
		if err != nil {
			return fmt.Errorf("could not parse date, the Budget.yfull file seems to be corrupt: %w", err)
		}

		accountImportHash := fmt.Sprint(sha256.Sum256([]byte(transaction.AccountID)))

		// Envelope Zero does not use a magic “Starting Balance” account, instead
		// every account has a field for the starting balance
		if accountIDNames[transaction.PayeeID] == "Starting Balance" {
			idx := slices.IndexFunc(resources.Accounts, func(a models.Account) bool {
				return a.ImportHash == accountImportHash
			})

			resources.Accounts[idx].InitialBalance = transaction.Amount
			resources.Accounts[idx].InitialBalanceDate = &date

			// Initial balance is set, no more processing needed
			continue
		}

		newTransaction := types.Transaction{
			Model: models.Transaction{
				TransactionCreate: models.TransactionCreate{
					Date:       date,
					Note:       strings.TrimSpace(transaction.Memo),
					ImportHash: fmt.Sprint(sha256.Sum256([]byte(transaction.EntityID))),
				},
			},
		}

		if transaction.Amount.IsPositive() {
			newTransaction.DestinationAccountHash = accountImportHash
			newTransaction.SourceAccountHash = payeeImportHash
			newTransaction.Model.Amount = transaction.Amount
		} else {
			newTransaction.SourceAccountHash = accountImportHash
			newTransaction.DestinationAccountHash = payeeImportHash
			newTransaction.Model.Amount = transaction.Amount.Neg()
		}

		// Set the reconciled flags
		if transaction.Cleared == "Reconciled" {
			if transaction.Amount.IsNegative() {
				newTransaction.Model.TransactionCreate.ReconciledSource = true
			} else {
				newTransaction.Model.TransactionCreate.ReconciledDestination = true
			}
		}

		// If the transaction is a transfer, we need to set ReconciledDestination
		if transaction.TargetAccountID != "" {
			// We find the corresponding transaction with the TransferTransactionID
			idx := slices.IndexFunc(transactions, func(t Transaction) bool { return t.EntityID == transaction.TransferTransactionID })
			if idx == -1 {
				return errors.New("could not find corresponding transaction, the Budget.yfull file seems to be corrupt")
			}

			// Depending on the transaction direction from the perspective of the current account, we need
			// to set which Reconciled flag we set.
			if transactions[idx].Cleared == "Reconciled" {
				if transaction.Amount.IsNegative() {
					newTransaction.Model.TransactionCreate.ReconciledDestination = true
				} else {
					newTransaction.Model.TransactionCreate.ReconciledSource = true
				}
			}
		}

		if transaction.CategoryID == "Category/__DeferredIncome__" {
			newTransaction.Model.AvailableFrom = internal_types.MonthOf(date).AddDate(0, 1)
		}

		// No subtransactions, add transaction directly
		if len(transaction.SubTransactions) == 0 {
			if mapping, ok := envelopeIDNames[transaction.CategoryID]; ok {
				newTransaction.Envelope = mapping.Envelope
				newTransaction.Category = mapping.Category
			}

			resources.Transactions = append(resources.Transactions, newTransaction)
			// Transaction has been added, nothing more to do
			continue
		}

		// Transaction has subtransactions, add them
		for _, sub := range transaction.SubTransactions {
			subTransaction := newTransaction

			if mapping, ok := envelopeIDNames[sub.CategoryID]; ok {
				subTransaction.Envelope = mapping.Envelope
				subTransaction.Category = mapping.Category
			}

			if sub.CategoryID == "Category/__DeferredIncome__" {
				subTransaction.Model.AvailableFrom = internal_types.MonthOf(date).AddDate(0, 1)
			} else {
				subTransaction.Model.AvailableFrom = internal_types.MonthOf(date)
			}

			// We need to set all of these again since the sub transaction can
			// have a positive or negative amount no matter what the amount of
			// the main transaction is
			if sub.Amount.IsPositive() {
				newTransaction.DestinationAccountHash = accountImportHash
				newTransaction.SourceAccountHash = payeeImportHash
				subTransaction.Model.Amount = sub.Amount
			} else {
				subTransaction.Model.Amount = sub.Amount.Neg()
				subTransaction.DestinationAccountHash = payeeImportHash
				subTransaction.SourceAccountHash = accountImportHash
			}

			// The transaction is a transfer
			if sub.TargetAccountID != "" {
				targetAccountImportHash := fmt.Sprint(sha256.Sum256([]byte(sub.TargetAccountID)))
				if sub.Amount.IsPositive() {
					subTransaction.SourceAccountHash = targetAccountImportHash
				} else {
					subTransaction.DestinationAccountHash = targetAccountImportHash
				}

				// We find the corresponding transaction with the TransferTransactionID
				idx := slices.IndexFunc(transactions, func(t Transaction) bool { return t.EntityID == sub.TransferTransactionID })
				if idx == -1 {
					return errors.New("could not find corresponding transaction for sub-transaction transfer, the Budget.yfull file seems to be corrupt")
				}

				// Depending on the transaction direction from the perspective of the current account, we need
				// to set which Reconciled flag we set.
				if transactions[idx].Cleared == "Reconciled" {
					if transaction.Amount.IsNegative() {
						subTransaction.Model.TransactionCreate.ReconciledDestination = true
					} else {
						subTransaction.Model.TransactionCreate.ReconciledSource = true
					}
				}
			}

			if sub.Memo != "" && subTransaction.Model.Note != "" {
				subTransaction.Model.Note = subTransaction.Model.Note + ": " + strings.TrimSpace(sub.Memo)
			} else if sub.Memo != "" {
				subTransaction.Model.Note = strings.TrimSpace(sub.Memo)
			}

			resources.Transactions = append(resources.Transactions, subTransaction)
		}
	}

	// Create the "no payee" payee if needed
	if addNoPayee {
		resources.Accounts = append(resources.Accounts, models.Account{
			AccountCreate: models.AccountCreate{
				Name:       "YNAB 4 Import - No Payee",
				Note:       "This is the opposing account for all transactions that were imported from YNAB 4, but did not have a Payee. In Envelope Zero, all transactions must have a Source and Destination account",
				OnBudget:   false,
				External:   true,
				ImportHash: noPayeeImportHash,
			},
		})
	}

	return nil
}

func parseMonthlyBudgets(resources *types.ParsedResources, monthlyBudgets []MonthlyBudget, envelopeIDNames IDToEnvelopes) error {
	for _, monthBudget := range monthlyBudgets {
		month, err := internal_types.ParseMonth(monthBudget.Month)
		if err != nil {
			return fmt.Errorf("could not parse date, the Budget.yfull file seems to be corrupt: %w", err)
		}

		for _, subCategoryBudget := range monthBudget.MonthlySubCategoryBudgets {
			// If the budget allocation is deleted, we don't need to do anything.
			// This is the case when a category that has budgeted amounts gets deleted.
			if subCategoryBudget.Deleted {
				continue
			}

			// If something is budgeted, create an allocation for it
			if !subCategoryBudget.Budgeted.IsZero() {
				resources.Allocations = append(resources.Allocations, types.Allocation{
					Model: models.Allocation{
						AllocationCreate: models.AllocationCreate{
							Month:  month,
							Amount: subCategoryBudget.Budgeted,
						},
					},
					Category: envelopeIDNames[subCategoryBudget.CategoryID].Category,
					Envelope: envelopeIDNames[subCategoryBudget.CategoryID].Envelope,
				})
			}

			// If the overspendHandling is configured, work with it
			if !(subCategoryBudget.OverspendingHandling == "") {
				// All occurrences of PreYNABDebt configurations that I could find are set for
				// months before there is any budget data.
				// Configuration for months before any data exists is not needed and therefore skipped
				//
				// If you find a budget where it is actually needed, please let me know!
				if strings.HasPrefix(subCategoryBudget.CategoryID, "Category/PreYNABDebt") {
					continue
				}

				var mode models.OverspendMode = "AFFECT_AVAILABLE"
				if subCategoryBudget.OverspendingHandling == "Confined" {
					mode = "AFFECT_ENVELOPE"
				}

				resources.MonthConfigs = append(resources.MonthConfigs, types.MonthConfig{
					Model: models.MonthConfig{
						MonthConfigCreate: models.MonthConfigCreate{
							OverspendMode: mode,
						},
						Month: month,
					},
					Category: envelopeIDNames[subCategoryBudget.CategoryID].Category,
					Envelope: envelopeIDNames[subCategoryBudget.CategoryID].Envelope,
				})
			}
		}
	}

	return nil
}

// fixOverspendHandling translates the overspend handling behaviour of YNAB 4 into
// the overspend handling of EZ. In YNAB 4, when the overspendHandling is set to "Confined",
// it affects all months until it is explicitly set back to "AffectsBuffer".
//
// EZ on the other hand uses AFFECT_AVAILABLE as default (as does YNAB 4 with "AffectsBuffer")
// but only changes to AFFECT_ENVELOPE (= "Confined" on YNAB 4) when explicitly configured for
// that month.
func fixOverspendHandling(resources *types.ParsedResources) {
	// sorter is a map of category names to a map of envelope names to the month configs
	sorter := make(map[string]map[string][]types.MonthConfig, 0)

	// Sort by envelope
	for _, monthConfig := range resources.MonthConfigs {
		_, ok := sorter[monthConfig.Category]
		if !ok {
			sorter[monthConfig.Category] = make(map[string][]types.MonthConfig, 0)
		}

		_, ok = sorter[monthConfig.Category][monthConfig.Envelope]
		if !ok {
			sorter[monthConfig.Category][monthConfig.Envelope] = make([]types.MonthConfig, 0)
		}

		sorter[monthConfig.Category][monthConfig.Envelope] = append(sorter[monthConfig.Category][monthConfig.Envelope], monthConfig)
	}

	// New slice for final MonthConfigs
	var monthConfigs []types.MonthConfig

	// Fix handling for all envelopes
	for _, category := range sorter {
		for _, monthConfig := range category {
			// Sort by time so that earlier months are first
			sort.Slice(monthConfig, func(i, j int) bool {
				return monthConfig[i].Model.Month.Before(monthConfig[j].Model.Month)
			})

			for i, mConfig := range monthConfig {
				// If we are switching back to "Available for budget", we don't need to do anything
				if mConfig.Model.OverspendMode == "AFFECT_AVAILABLE" || mConfig.Model.OverspendMode == "" {
					continue
				}

				monthConfigs = append(monthConfigs, mConfig)

				// Start with the next month since we already appended the current one
				checkMonth := mConfig.Model.Month.AddDate(0, 1)

				// If this is the last month, we set all months including the one of today to "AFFECT_ENVELOPE"
				// to preserve the YNAB 4 behaviour up to the switch to EZ
				if i+1 == len(monthConfig) {
					for ok := true; ok; ok = !checkMonth.AfterTime(time.Now()) {
						monthConfigs = append(monthConfigs, types.MonthConfig{
							Model: models.MonthConfig{
								Month: checkMonth,
								MonthConfigCreate: models.MonthConfigCreate{
									OverspendMode: models.AffectEnvelope,
								},
							},
							Category: mConfig.Category,
							Envelope: mConfig.Envelope,
						})

						checkMonth = checkMonth.AddDate(0, 1)
					}

					continue
				}

				// Set all months up to the next one with a configuration to "AFFECT_ENVELOPE"
				for ok := !checkMonth.Equal(monthConfig[i+1].Model.Month); ok; ok = !checkMonth.Equal(monthConfig[i+1].Model.Month) {
					monthConfigs = append(monthConfigs, types.MonthConfig{
						Model: models.MonthConfig{
							Month: checkMonth,
							MonthConfigCreate: models.MonthConfigCreate{
								OverspendMode: "AFFECT_ENVELOPE",
							},
						},
						Category: mConfig.Category,
						Envelope: mConfig.Envelope,
					})

					checkMonth = checkMonth.AddDate(0, 1)
				}
			}
		}
	}

	// Overwrite the original MonthConfigs with the fixed ones
	resources.MonthConfigs = monthConfigs
}
