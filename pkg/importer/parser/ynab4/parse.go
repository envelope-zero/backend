package ynab4

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/envelope-zero/backend/pkg/importer/types"
	"github.com/envelope-zero/backend/pkg/models"
	"golang.org/x/exp/maps"
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

	// Add all accounts
	accountIDNames, err := parseAccounts(&resources, budget.Accounts)
	if err != nil {
		return types.ParsedResources{}, fmt.Errorf("error parsing accounts: %w", err)
	}

	payeeIDNames, err := parsePayees(&resources, budget.Payees)
	if err != nil {
		return types.ParsedResources{}, fmt.Errorf("error parsing payees: %w", err)
	}

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

	err = parseAllocations(&resources, budget.MonthlyBudgets, envelopeIDNames)
	if err != nil {
		return types.ParsedResources{}, fmt.Errorf("error parsing budget allocations: %w", err)
	}

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

func parseAccounts(resources *types.ParsedResources, accounts []Account) (IDToName, error) {
	idToNames := make(IDToName)

	resources.Accounts = make(map[string]types.Account)
	for _, account := range accounts {
		idToNames[account.EntityID] = account.Name
		resources.Accounts[account.Name] = types.Account{
			Model: models.Account{
				AccountCreate: models.AccountCreate{
					Name:     account.Name,
					Note:     account.Note,
					OnBudget: account.OnBudget,
					Hidden:   account.Hidden,
				},
			},
		}
	}

	return idToNames, nil
}

func parsePayees(resources *types.ParsedResources, payees []Payee) (IDToName, error) {
	idToNames := make(IDToName)

	// Payees in YNAB 4 map to External Accounts in Envelope Zero
	for _, payee := range payees {
		if payee.Deleted {
			continue
		}

		// Transfers are also stored as Payees with an entity ID of "Payee/Transfer:[Target account ID]"
		// As we do not need this hack for Envelope Zero, we skip those Payees
		if strings.HasPrefix(payee.EntityID, "Payee/Transfer") {
			continue
		}

		// Create the account
		idToNames[payee.EntityID] = payee.Name
		resources.Accounts[payee.Name] = types.Account{
			Model: models.Account{
				AccountCreate: models.AccountCreate{
					Name:     payee.Name,
					OnBudget: false,
					External: true,
				},
			},
		}
	}

	return idToNames, nil
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
					return IDToEnvelopes{}, fmt.Errorf("hidden category could not be parsed: %w", err)
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
		category, ok := tCategories[envelope.Category]
		if !ok {
			return IDToEnvelopes{}, errors.New("an envelope referenced a non-existing category. Your Budget.yfull file seems to be inconsistent")
		}

		// Check if the category already exists in the resources. If not, create it
		_, ok = resources.Categories[envelope.Category]
		if !ok {
			resources.Categories[envelope.Category] = category
		}

		resources.Categories[envelope.Category].Envelopes[envelope.Envelope.Model.Name] = envelope.Envelope
	}

	return idToEnvelope, nil
}

func parseTransactions(resources *types.ParsedResources, transactions []Transaction, accountIDNames IDToName, envelopeIDNames IDToEnvelopes) error {
	// If an account "No payee" for transactions without a payee needs to be added
	addNoPayee := false

	// Add all transactions
	for _, transaction := range transactions {
		// Don't import deleted transactions or transactions that are 0
		//
		// Transfers create two corresponding transaction entries in YNAB 4
		//
		// They use the same entityId, but one is suffixed with "_T_0"
		// Therefore, we ignore transactions where the entity ID ends in
		// "_T_0"
		if transaction.Deleted || transaction.Amount.IsZero() || strings.HasSuffix(transaction.EntityID, "_T_0") {
			continue
		}

		// For transactions, the payee string has the prefix "Payee/Transfer:"
		payeeID := strings.TrimPrefix(transaction.PayeeID, "Payee/Transfer:")

		// If we do not have a Payee for a transaction, we use the special import payee/account
		// that will be created only if it is needed
		payee := accountIDNames[payeeID]
		if payee == "" {
			payee = "YNAB 4 Import - No Payee"
			addNoPayee = true
		}

		// Envelope Zero does not use a magic “Starting Balance” account, instead
		// every account has a field for the starting balance
		if payee == "Starting Balance" {
			account := resources.Accounts[accountIDNames[transaction.AccountID]]
			account.Model.InitialBalance = transaction.Amount

			resources.Accounts[accountIDNames[transaction.AccountID]] = account

			// Initial balance is set, no more processing needed
			continue
		}

		// Parse the date of the transaction
		date, err := time.Parse("2006-01-02", transaction.Date)
		if err != nil {
			return fmt.Errorf("could not parse date, the Budget.yfull file seems to be corrupt: %w", err)
		}

		newTransaction := types.Transaction{
			Model: models.Transaction{
				TransactionCreate: models.TransactionCreate{
					Date: date,
					Note: transaction.Memo,
				},
			},
		}

		if transaction.Amount.IsPositive() {
			newTransaction.DestinationAccount = accountIDNames[transaction.AccountID]
			newTransaction.SourceAccount = payee
			newTransaction.Model.Amount = transaction.Amount
		} else {
			newTransaction.SourceAccount = accountIDNames[transaction.AccountID]
			newTransaction.DestinationAccount = payee
			newTransaction.Model.Amount = transaction.Amount.Neg()
		}

		if transaction.Cleared == "Reconciled" {
			newTransaction.Model.TransactionCreate.Reconciled = true
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
			if mapping, ok := envelopeIDNames[sub.CategoryID]; ok {
				newTransaction.Envelope = mapping.Envelope
				newTransaction.Category = mapping.Category
			}
			newTransaction.Model.Amount = sub.Amount

			resources.Transactions = append(resources.Transactions, newTransaction)
		}
	}

	if addNoPayee {
		if _, ok := resources.Accounts["YNAB 4 Import - No Payee"]; !ok {
			resources.Accounts["YNAB 4 Import - No Payee"] = types.Account{
				Model: models.Account{
					AccountCreate: models.AccountCreate{
						Name:     "YNAB 4 Import - No Payee",
						Note:     "This is the opposing account for all transactions that were imported from YNAB 4, but did not have a Payee. In Envelope Zero, all transactions must have a Source and Destination account",
						OnBudget: false,
						External: true,
					},
				},
			}
		}
	}

	return nil
}

func parseAllocations(resources *types.ParsedResources, allocations []MonthlyBudget, envelopeIDNames IDToEnvelopes) error {
	for _, monthBudget := range allocations {
		month, err := time.Parse("2006-01-02", monthBudget.Month)
		if err != nil {
			return fmt.Errorf("could not parse date: %w", err)
		}

		for _, allocation := range monthBudget.MonthlySubCategoryBudgets {
			// Ignore the ones that are zero
			if allocation.Budgeted.IsZero() {
				continue
			}

			resources.Allocations = append(resources.Allocations, types.Allocation{
				Model: models.Allocation{
					AllocationCreate: models.AllocationCreate{
						Month:  month,
						Amount: allocation.Budgeted,
					},
				},
				Category: envelopeIDNames[allocation.CategoryID].Category,
				Envelope: envelopeIDNames[allocation.CategoryID].Envelope,
			})
		}
	}

	return nil
}
