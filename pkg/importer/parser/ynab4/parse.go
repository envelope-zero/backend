package ynab4

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/google/uuid"

	"github.com/envelope-zero/backend/v4/pkg/importer"
	"github.com/envelope-zero/backend/v4/pkg/importer/helpers"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/text/currency"
)

// This function parses a YNAB 4 Budget.yfull file.
func Parse(f io.Reader) (importer.ParsedResources, error) {
	content, err := io.ReadAll(f)
	if err != nil {
		return importer.ParsedResources{}, fmt.Errorf("could not read data from file: %w", err)
	}

	var budget Budget
	err = json.Unmarshal(content, &budget)
	if err != nil {
		return importer.ParsedResources{}, fmt.Errorf("not a valid YNAB4 Budget.yfull file: %w", err)
	}

	var resources importer.ParsedResources

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
		return importer.ParsedResources{}, fmt.Errorf("error parsing categories and subcategories: %w", err)
	}

	err = parseTransactions(&resources, budget.Transactions, accountIDNames, envelopeIDNames)
	if err != nil {
		return importer.ParsedResources{}, fmt.Errorf("error parsing transactions: %w", err)
	}

	err = parseMonthlyBudgets(&resources, budget.MonthlyBudgets, envelopeIDNames)
	if err != nil {
		return importer.ParsedResources{}, fmt.Errorf("error parsing budget allocations: %w", err)
	}

	generateOverspendFixes(&resources)

	// Fix duplicate account names
	fixDuplicateAccountNames(&resources)

	return resources, nil
}

func parseArchivedCategoryName(f string) (category, envelope string, err error) {
	// The format of archived category strings is shown in the next line. Square brackets denote field names
	// [Master Category Name] ` [Category Name] ` [Archival Number]
	match := regexp.MustCompile("(.*) ` (.*) `").FindStringSubmatch(f)

	// len needs to be 3 as the whole regex match is in match[0]
	if len(match) != 3 {
		return "", "", fmt.Errorf("incorrect archived category format: match length is %d", len(match))
	}

	category = match[1]
	envelope = match[2]
	return
}

func parseAccounts(resources *importer.ParsedResources, accounts []Account) IDToName {
	idToNames := make(IDToName)

	for _, account := range accounts {
		idToNames[account.EntityID] = account.Name

		resources.Accounts = append(resources.Accounts, models.Account{
			AccountCreate: models.AccountCreate{
				Name:       account.Name,
				Note:       account.Note,
				OnBudget:   account.OnBudget,
				Archived:   account.Archived,
				ImportHash: helpers.Sha256String(account.EntityID),
			},
		})
	}

	return idToNames
}

func parsePayees(resources *importer.ParsedResources, payees []Payee) IDToName {
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
				ImportHash: helpers.Sha256String(payee.EntityID),
			},
		})

		// Parse the Match Rules from the payee's rename conditions
		for _, r := range payee.RenameConditions {
			// Skip deleted rename conditions
			if r.Deleted {
				continue
			}

			// Determine the match string. Since EZ uses globs and YNAB4 has different
			// operators, we translate between the two
			var match string
			switch r.Operator {
			case "Is":
				match = r.Operand
			case "Contains":
				match = fmt.Sprintf("*%s*", r.Operand)
			case "StartsWith":
				match = fmt.Sprintf("%s*", r.Operand)
			case "EndsWith":
				match = fmt.Sprintf("*%s", r.Operand)
			}

			resources.MatchRules = append(resources.MatchRules, importer.MatchRule{
				Account: payee.Name,
				MatchRule: models.MatchRule{
					MatchRuleCreate: models.MatchRuleCreate{
						Priority: 0,
						Match:    match,
					},
				},
			})
		}
	}

	return idToNames
}

func parseCategories(resources *importer.ParsedResources, categories []Category) (IDToEnvelopes, error) {
	idToEnvelope := make(IDToEnvelopes)

	// Create temporary variables to hold all the parsed
	// data. They will be added to the ParsedResources
	// when parsing is complete.
	tCategories := make(map[string]importer.Category)
	type tEnvelope struct {
		Envelope importer.Envelope
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
		tCategories[category.Name] = importer.Category{
			Model: models.Category{
				CategoryCreate: models.CategoryCreate{
					Name: category.Name,
					Note: category.Note,
					// we use category.Deleted here since the original data format does not have an "archived" field. If the category is not referenced anywhere,
					// it will not be imported anyway
					Archived: category.Deleted,
				},
			},
			Envelopes: make(map[string]importer.Envelope),
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

			// For archived categories, we need to extract the actual name
			var archived bool
			if category.Name == "Hidden Categories" {
				var err error
				mapping.Category, mapping.Envelope, err = parseArchivedCategoryName(mapping.Envelope)
				if err != nil {
					return IDToEnvelopes{}, fmt.Errorf("hidden category could not be parsed, your Budget.yfull file seems to be corrupted: %w", err)
				}

				archived = true
			}

			idToEnvelope[envelope.EntityID] = mapping

			tEnvelopes = append(tEnvelopes, tEnvelope{
				importer.Envelope{
					Model: models.Envelope{
						EnvelopeCreate: models.EnvelopeCreate{
							Name:     mapping.Envelope,
							Note:     envelope.Note,
							Archived: archived,
						},
					},
				},
				mapping.Category,
			})
		}
	}

	// Initialize the categories
	resources.Categories = make(map[string]importer.Category)

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

func parseTransactions(resources *importer.ParsedResources, transactions []Transaction, accountIDNames IDToName, envelopeIDNames IDToEnvelopes) error {
	// If an account "No payee" for transactions without a payee needs to be added
	addNoPayee := false

	// We generate an import hash for the "YNAB 4 - No payee" account that might be added since
	// we create it and it therefore does not have a UUID
	noPayeeImportHash := helpers.Sha256String(uuid.New().String())

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
			payeeImportHash = helpers.Sha256String(transaction.PayeeID)
		}

		// Parse the date of the transaction
		date, err := time.Parse("2006-01-02", transaction.Date)
		if err != nil {
			return fmt.Errorf("could not parse date, the Budget.yfull file seems to be corrupt: %w", err)
		}

		accountImportHash := helpers.Sha256String(transaction.AccountID)

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

		newTransaction := importer.Transaction{
			Model: models.Transaction{
				TransactionCreate: models.TransactionCreate{
					Date:       date,
					Note:       strings.TrimSpace(transaction.Memo),
					ImportHash: helpers.Sha256String(transaction.EntityID),
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
			newTransaction.Model.AvailableFrom = types.MonthOf(date).AddDate(0, 1)
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
				subTransaction.Model.AvailableFrom = types.MonthOf(date).AddDate(0, 1)
			} else {
				subTransaction.Model.AvailableFrom = types.MonthOf(date)
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
				targetAccountImportHash := helpers.Sha256String(sub.TargetAccountID)
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

func parseMonthlyBudgets(resources *importer.ParsedResources, monthlyBudgets []MonthlyBudget, envelopeIDNames IDToEnvelopes) error {
	slices.SortFunc(monthlyBudgets, func(a, b MonthlyBudget) int {
		if a.Month.Before(b.Month) {
			return -1
		}

		if b.Month.Before(a.Month) {
			return 1
		}

		return 0
	})

	for _, monthBudget := range monthlyBudgets {
		for _, subCategoryBudget := range monthBudget.MonthlySubCategoryBudgets {
			// If the budget allocation is deleted, we don't need to do anything.
			// This is the case when a category that has budgeted amounts gets deleted.
			//
			// Category/PreYNABDebt: All occurrences of PreYNABDebt configurations that I could find are set for
			// months before there is any budget data.
			// Configuration for months before any data exists is not needed and therefore skipped
			//
			// If you find a budget where it is actually needed, please let me know!
			if subCategoryBudget.Deleted || strings.HasPrefix(subCategoryBudget.CategoryID, "Category/PreYNABDebt") {
				continue
			}

			monthConfig := importer.MonthConfig{
				Model: models.MonthConfig{
					Month: monthBudget.Month,
					MonthConfigCreate: models.MonthConfigCreate{
						Allocation: subCategoryBudget.Budgeted,
					},
				},
				Category: envelopeIDNames[subCategoryBudget.CategoryID].Category,
				Envelope: envelopeIDNames[subCategoryBudget.CategoryID].Envelope,
			}

			// If the overspendHandling is configured, work with it
			if subCategoryBudget.OverspendingHandling != "" {
				monthConfig.OverspendMode = importer.AffectAvailable
				if subCategoryBudget.OverspendingHandling == "Confined" {
					monthConfig.OverspendMode = importer.AffectEnvelope
				}
			}

			resources.MonthConfigs = append(resources.MonthConfigs, monthConfig)
		}
	}

	return nil
}

// fixDuplicateAccountNames detects if an account name is the same for an internal and
// external account (which is allowed in YNAB for accounts and Payees) and adds
// " (External)" to the external (payee) account.
func fixDuplicateAccountNames(r *importer.ParsedResources) {
	for i := 0; i < len(r.Accounts); i++ {
		// Loop over all accounts later in the list
		for j := i + 1; j < len(r.Accounts); j++ {
			// If the accounts names match, rename the external account
			if r.Accounts[j].Name == r.Accounts[i].Name {
				var a *models.Account

				if r.Accounts[i].External {
					a = &r.Accounts[i]
				} else {
					a = &r.Accounts[j]
				}

				a.Name = fmt.Sprintf("%s (External)", a.Name)
			}
		}
	}
}

// generateOverspendFixes translates the overspend handling behaviour of YNAB 4 into
// the overspend handling of EZ. In YNAB 4, when the overspendHandling is set to "Confined",
// it affects all months until it is explicitly set back to "AffectsBuffer".
//
// Envelope Zero does not support overspend handling, so we generate an OverspendFix for every
// month that is affected by "Confined" overspend handling in YNAB4.
//
// The OverspendFixes then will be used by the creator to correctly update allocations to envelopes
// to preserve the correct budgeted values
func generateOverspendFixes(resources *importer.ParsedResources) {
	// sorter is a map of category names to a map of envelope names to the month configs
	sorter := make(map[string]map[string][]importer.MonthConfig, 0)

	// Sort by envelope
	for _, monthConfig := range resources.MonthConfigs {
		_, ok := sorter[monthConfig.Category]
		if !ok {
			sorter[monthConfig.Category] = make(map[string][]importer.MonthConfig, 0)
		}

		_, ok = sorter[monthConfig.Category][monthConfig.Envelope]
		if !ok {
			sorter[monthConfig.Category][monthConfig.Envelope] = make([]importer.MonthConfig, 0)
		}

		sorter[monthConfig.Category][monthConfig.Envelope] = append(sorter[monthConfig.Category][monthConfig.Envelope], monthConfig)
	}

	// Fix handling for all envelopes
	for _, envelopes := range sorter {
		for _, monthConfigs := range envelopes {
			// Sort by time so that earlier months are first
			sort.Slice(monthConfigs, func(i, j int) bool {
				return monthConfigs[i].Model.Month.Before(monthConfigs[j].Model.Month)
			})

			for i, monthConfig := range monthConfigs {
				// If we are switching back to "Available for budget", we don't need to do anything
				// anymore and can go to the next month config
				if monthConfig.OverspendMode == importer.AffectAvailable {
					continue
				}

				// Append an overspend fix for this month
				resources.OverspendFixes = append(resources.OverspendFixes, importer.OverspendFix{
					Category: monthConfig.Category,
					Envelope: monthConfig.Envelope,
					Month:    monthConfig.Model.Month,
				})

				// Start with the next month since we already appended
				// an overspend fix for the current one
				checkMonth := monthConfig.Model.Month.AddDate(0, 1)

				// If the checkMonth is the last month for which we have a month config,
				// we then add overspend fixes for all month up until the date at which
				// the import happens.
				//
				// This is done so that the budget values are exactly the same up until
				// the month where the users is importing to Envelope Zero.
				//
				// This enables users to compare the values and verify they are the same.
				if i+1 == len(monthConfigs) {
					for ok := true; ok; ok = !checkMonth.AfterTime(time.Now()) {
						resources.OverspendFixes = append(resources.OverspendFixes, importer.OverspendFix{
							Category: monthConfig.Category,
							Envelope: monthConfig.Envelope,
							Month:    checkMonth,
						})

						checkMonth = checkMonth.AddDate(0, 1)
					}
					continue
				}

				// Set all months up to the next one with a configuration to "AFFECT_ENVELOPE"
				// We have not arrived at the last month config in the list, so we add overspend
				// fixes for the current month and all months that are between the current month
				// and the next month for which we have a month config
				for ok := !checkMonth.Equal(monthConfigs[i+1].Model.Month); ok; ok = !checkMonth.Equal(monthConfigs[i+1].Model.Month) {
					resources.OverspendFixes = append(resources.OverspendFixes, importer.OverspendFix{
						Category: monthConfig.Category,
						Envelope: monthConfig.Envelope,
						Month:    checkMonth,
					})

					checkMonth = checkMonth.AddDate(0, 1)
				}
			}
		}
	}
}
