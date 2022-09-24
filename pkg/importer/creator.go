package importer

import (
	"github.com/envelope-zero/backend/pkg/importer/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Create(db *gorm.DB, budgetName string, resources types.ParsedResources) error {
	// Start a transaction so we can roll back all created resources if an error occurs
	tx := db.Begin()

	// Create the budget
	budget := resources.Budget
	budget.BudgetCreate.Name = budgetName
	err := tx.Create(&budget).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// Create accounts
	for name, r := range resources.Accounts {
		// Set the Account ID & name and create the account
		account := r.Model
		account.AccountCreate.BudgetID = budget.ID
		account.AccountCreate.Name = name
		err := tx.Create(&account).Error
		if err != nil {
			tx.Rollback()
			return err
		}

		// Update the account in the resources struct so that it also contains the ID
		r.Model = account
		resources.Accounts[name] = r
	}

	for cName, category := range resources.Categories {
		category.Model.BudgetID = budget.ID

		err := tx.Create(&category.Model).Error
		if err != nil {
			tx.Rollback()
			return err
		}
		resources.Categories[cName] = category

		// Add all envelopes
		for eName, envelope := range category.Envelopes {
			envelope.Model.CategoryID = category.Model.ID

			err := tx.Create(&envelope.Model).Error
			if err != nil {
				tx.Rollback()
				return err
			}
			resources.Categories[category.Model.Name].Envelopes[eName] = envelope
		}
	}

	// Create transactions
	for _, r := range resources.Transactions {
		transaction := r.Model
		transaction.BudgetID = budget.ID
		transaction.SourceAccountID = resources.Accounts[r.SourceAccount].Model.ID
		transaction.DestinationAccountID = resources.Accounts[r.DestinationAccount].Model.ID

		envelopeID := resources.Categories[r.Category].Envelopes[r.Envelope].Model.ID
		if envelopeID != uuid.Nil {
			transaction.EnvelopeID = &envelopeID
		}

		err := tx.Create(&transaction).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, a := range resources.Allocations {
		allocation := a.Model
		allocation.AllocationCreate.EnvelopeID = resources.Categories[a.Category].Envelopes[a.Envelope].Model.ID

		err := tx.Create(&allocation).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// No errors happened, commit the transaction
	tx.Commit()
	return nil
}
