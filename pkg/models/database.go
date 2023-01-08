package models

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate migrates all models to the schema defined in the code.
func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, Allocation{}, MonthConfig{})
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
	}

	/*
	 * Workaround for https://github.com/go-gorm/gorm/issues/5968
	 */
	// Account
	db.Unscoped().Model(&Account{}).Select("OnBudget").Where("accounts.on_budget IS NULL").Update("OnBudget", false)
	db.Unscoped().Model(&Account{}).Select("External").Where("accounts.external IS NULL").Update("External", false)
	db.Unscoped().Model(&Account{}).Select("Hidden").Where("accounts.hidden IS NULL").Update("Hidden", false)

	// Category
	db.Unscoped().Model(&Category{}).Select("Hidden").Where("categories.hidden IS NULL").Update("Hidden", false)

	// Envelope
	db.Unscoped().Model(&Envelope{}).Select("Hidden").Where("envelopes.hidden IS NULL").Update("Hidden", false)

	// Transaction
	db.Unscoped().Model(&Transaction{}).Select("Reconciled").Where("transactions.reconciled IS NULL").Update("Reconciled", false)
	db.Unscoped().Model(&Transaction{}).Select("ReconciledSource").Where("transactions.reconciled_source IS NULL").Update("ReconciledSource", false)
	db.Unscoped().Model(&Transaction{}).Select("ReconciledDestination").Where("transactions.reconciled_destination IS NULL").Update("ReconciledDestination", false)

	return nil
}
