package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migrate migrates all models to the schema defined in the code.
func Migrate(db *gorm.DB) (err error) {
	// Migration for https://github.com/envelope-zero/backend/issues/684
	// Remove with 4.0.0
	// This migration is done before the AutoMigrate since AutoMigrate will introduce
	// new unique constraints that this migration ensures are fulfilled for existing
	// transactions
	if db.Migrator().HasTable(&Account{}) {
		err = migrateDuplicateAccountNames(db)
		if err != nil {
			return fmt.Errorf("error during migrateDuplicateAccountNames: %w", err)
		}
	}

	// https://github.com/envelope-zero/backend/issues/763
	// Remove with 4.0.0
	if db.Migrator().HasTable("rename_rules") {
		err := db.Migrator().RenameTable("rename_rules", "match_rules")
		if err != nil {
			return fmt.Errorf("error during rename_rules -> match_rules migration: %w", err)
		}
	}

	err = db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, Allocation{}, MonthConfig{}, MatchRule{})
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
	}

	// Migration for https://github.com/envelope-zero/backend/issues/613
	// Remove with 4.0.0
	err = unsetEnvelopes(db)
	if err != nil {
		return fmt.Errorf("error during unsetEnvelopes: %w", err)
	}

	return nil
}

// migrateDuplicateAccountNames migrates duplicate account names to be unique.
func migrateDuplicateAccountNames(db *gorm.DB) (err error) {
	type Duplicate struct {
		BudgetID uuid.UUID
		Name     string
	}

	// Get a list of budget ID and account name for all budgets that have duplicate account names
	var duplicates []Duplicate
	err = db.Raw("select budget_id, name, COUNT(*) from accounts GROUP BY budget_id, name having count(*) > 1").Scan(&duplicates).Error
	if err != nil {
		return
	}

	for _, d := range duplicates {
		var accounts []Account

		// Find all accounts that have a duplicate name
		err = db.Unscoped().Where(Account{AccountCreate: AccountCreate{
			BudgetID: d.BudgetID,
			Name:     d.Name,
		}}).Find(&accounts).Error
		if err != nil {
			return
		}

		for i, a := range accounts {
			a.Name = fmt.Sprintf("%s (%d)", a.Name, i+1)
			err = db.Save(&a).Error
			if err != nil {
				return
			}
		}
	}

	return nil
}

// unsetEnvelopes removes the envelopes from transfers between
// accounts that are on budget.
func unsetEnvelopes(db *gorm.DB) (err error) {
	var accounts []Account
	err = db.Where(&Account{AccountCreate: AccountCreate{
		OnBudget: true,
	}}).Find(&accounts).Error
	if err != nil {
		return
	}

	for _, a := range accounts {
		var transactions []Transaction
		err = db.Model(&Transaction{}).
			Joins("JOIN accounts ON transactions.source_account_id = accounts.id").
			Where("destination_account_id = ? AND accounts.on_budget AND envelope_id not null", a.ID).
			Find(&transactions).Error
		if err != nil {
			return
		}

		for _, t := range transactions {
			err = db.Model(&t).Select("EnvelopeID").Updates(map[string]interface{}{"EnvelopeID": nil}).Error
			if err != nil {
				return
			}
		}
	}

	return
}
