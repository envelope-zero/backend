package importer

import (
	"errors"
	"fmt"

	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
		account.BudgetID = budget.ID
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

	for _, f := range resources.OverspendFixes {
		envelopeID := resources.Categories[f.Category].Envelopes[f.Envelope].Model.ID

		var envelope models.Envelope
		err := tx.First(&envelope, envelopeID).Error
		if err != nil {
			tx.Rollback()
			return models.Budget{}, fmt.Errorf("could not find envelope to fix overspend on: %w", err)
		}

		balance, err := envelope.Balance(tx, f.Month)
		if err != nil {
			tx.Rollback()
			return models.Budget{}, fmt.Errorf("error on balance calculation for envelope to fix overspend on: %w", err)
		}

		// If the envelope is not overspent (i.e. balance is >= 0), we don't need to do anything
		if balance.GreaterThanOrEqual(decimal.Zero) {
			continue
		}

		// We need to add(!) the envelope balance to the allocation for the next month.
		// To do so, we find the MonthConfig or create it
		var monthConfig models.MonthConfig
		err = tx.Where(models.MonthConfig{
			Month:      f.Month.AddDate(0, 1),
			EnvelopeID: envelopeID,
		}).FirstOrCreate(&monthConfig).Error
		if err != nil {
			tx.Rollback()
			return models.Budget{}, fmt.Errorf("error on reading/creating the month config for overspend fixing: %w", err)
		}

		// Add the balance
		// We need to subtract the overspent amount, since the balance is negative the overspent amount, we add it
		monthConfig.Allocation = monthConfig.Allocation.Add(balance)
		err = tx.Save(&monthConfig).Error
		if err != nil {
			tx.Rollback()
			return models.Budget{}, fmt.Errorf("error on updating the month config for overspend fixing: %w", err)
		}
	}

	// No errors happened, commit the transaction
	tx.Commit()
	return budget, nil
}
