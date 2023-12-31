package importer

import (
	"errors"
	"fmt"

	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

func Create(db *gorm.DB, resources ParsedResources) (models.Budget, error) {
	// Start a transaction so we can roll back all created resources if an error occurs
	tx := db.Begin()

	// Create the budget
	budget := resources.Budget
	err := tx.Create(&budget).Error
	if err != nil {
		tx.Rollback()
		return models.Budget{}, err
	}

	// Create accounts
	for idx, account := range resources.Accounts {
		account.AccountCreate.BudgetID = budget.ID
		err := tx.Create(&account).Error
		if err != nil {
			tx.Rollback()
			return models.Budget{}, err
		}

		// Update the account in the resources struct so that it also contains the ID
		resources.Accounts[idx] = account
	}

	// Create Match Rules
	for _, matchRule := range resources.MatchRules {
		aIdx := slices.IndexFunc(resources.Accounts, func(a models.Account) bool { return a.Name == matchRule.Account })
		if aIdx == -1 {
			tx.Rollback()
			return models.Budget{}, fmt.Errorf("the account '%s' specified in the Match Rule matching '%s' could not be found in the list of Accounts", matchRule.Account, matchRule.Match)
		}

		matchRule.MatchRule.AccountID = resources.Accounts[aIdx].ID

		err := tx.Create(&matchRule.MatchRule).Error
		if err != nil {
			tx.Rollback()
			return models.Budget{}, err
		}
	}

	for cName, category := range resources.Categories {
		category.Model.BudgetID = budget.ID

		err := tx.Create(&category.Model).Error
		if err != nil {
			tx.Rollback()
			return models.Budget{}, err
		}
		resources.Categories[cName] = category

		// Add all envelopes
		for eName, envelope := range category.Envelopes {
			envelope.Model.CategoryID = category.Model.ID

			err := tx.Create(&envelope.Model).Error
			if err != nil {
				tx.Rollback()
				return models.Budget{}, err
			}
			resources.Categories[category.Model.Name].Envelopes[eName] = envelope
		}
	}

	// Create transactions
	for _, r := range resources.Transactions {
		if r.Model.Amount.IsNegative() {
			return models.Budget{}, errors.New("a transaction to be imported has a negative amount, this is invalid")
		}

		transaction := r.Model
		transaction.BudgetID = budget.ID

		// Find the source account and set it
		idx := slices.IndexFunc(resources.Accounts, func(a models.Account) bool {
			return a.ImportHash == r.SourceAccountHash
		})
		transaction.SourceAccountID = resources.Accounts[idx].ID

		// Find the destination account and set it
		idx = slices.IndexFunc(resources.Accounts, func(a models.Account) bool {
			return a.ImportHash == r.DestinationAccountHash
		})
		transaction.DestinationAccountID = resources.Accounts[idx].ID

		envelopeID := resources.Categories[r.Category].Envelopes[r.Envelope].Model.ID
		if envelopeID != uuid.Nil {
			transaction.EnvelopeID = &envelopeID
		}

		err := tx.Create(&transaction).Error
		if err != nil {
			tx.Rollback()
			return models.Budget{}, err
		}
	}

	// Create MonthConfigs
	for i, m := range resources.MonthConfigs {
		mConfig := m.Model
		mConfig.EnvelopeID = resources.Categories[m.Category].Envelopes[m.Envelope].Model.ID

		err := tx.Create(&mConfig).Error
		if err != nil {
			tx.Rollback()
			return models.Budget{}, fmt.Errorf("error on creation of month config %d: %w", i, err)
		}
	}

	// No errors happened, commit the transaction
	tx.Commit()
	return budget, nil
}
