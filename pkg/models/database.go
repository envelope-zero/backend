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

	err = db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, Allocation{}, MonthConfig{}, RenameRule{})
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
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
