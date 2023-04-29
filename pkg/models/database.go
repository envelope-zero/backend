package models

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Migrate migrates all models to the schema defined in the code.
func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, Allocation{}, MonthConfig{})
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
	}

	queries := []*gorm.DB{
		/*
		 * Workaround for https://github.com/go-gorm/gorm/issues/5968
		 * Remove with 3.0.0
		 */
		// Account
		db.Unscoped().Model(&Account{}).Select("OnBudget").Where("accounts.on_budget IS NULL").Update("OnBudget", false),
		db.Unscoped().Model(&Account{}).Select("External").Where("accounts.external IS NULL").Update("External", false),
		db.Unscoped().Model(&Account{}).Select("Hidden").Where("accounts.hidden IS NULL").Update("Hidden", false),
		// Category
		db.Unscoped().Model(&Category{}).Select("Hidden").Where("categories.hidden IS NULL").Update("Hidden", false),
		// Envelope
		db.Unscoped().Model(&Envelope{}).Select("Hidden").Where("envelopes.hidden IS NULL").Update("Hidden", false),
		// Transaction
		db.Unscoped().Model(&Transaction{}).Select("Reconciled").Where("transactions.reconciled IS NULL").Update("Reconciled", false),
		db.Unscoped().Model(&Transaction{}).Select("ReconciledSource").Where("transactions.reconciled_source IS NULL").Update("ReconciledSource", false),
		db.Unscoped().Model(&Transaction{}).Select("ReconciledDestination").Where("transactions.reconciled_destination IS NULL").Update("ReconciledDestination", false),

		// Delete allocations with an amount of 0
		db.Unscoped().Model(&Allocation{}).Where("amount IS '0'").Delete(&Allocation{}),
	}
	for _, query := range queries {
		err = query.Error
		if err != nil {
			return fmt.Errorf("error during DB migration: %w", err)
		}
	}

	/*
	 * Complex migrations
	 */

	// Migration for https://github.com/envelope-zero/backend/issues/628.
	// Remove with 3.0.0
	err = migrateImportHashString(db)
	if err != nil {
		return fmt.Errorf("error during migrateImportHashString: %w", err)
	}

	return nil
}

// migrateImportHashString migrates the string representation of the SHA256 hash as byte array
// to a hex string representation of the hash to use the common way of representing SHA256 hashes.
// See https://github.com/envelope-zero/backend/issues/628.
func migrateImportHashString(db *gorm.DB) (err error) {
	var accounts []Account
	err = db.Unscoped().Where("import_hash LIKE '[%'").Find(&accounts).Error
	if err != nil {
		return err
	}

	for _, account := range accounts {
		// The string looks like this: "[40 52 207 7 118 61 80 107 178 242 5 47 211 161 180 135 104 222 118 28 56 12 33 63 179 78 39 173 206 11 77 3]"
		// With trimming and splitting it, we get a slice containing every individual number
		bytes := strings.Split(strings.TrimRight(strings.TrimLeft(account.ImportHash, "["), "]"), " ")

		// Assemble the slice back to a string. We pad with zeroes so that every byte takes two characters
		var b strings.Builder
		for _, part := range bytes {
			// Need to convert to int first so that it's interpreted correctly
			charAsInt, _ := strconv.Atoi(part)
			b.WriteString(fmt.Sprintf("%02x", byte(charAsInt)))
		}

		// Save the record back to the DB
		account.ImportHash = b.String()
		db.Unscoped().Save(&account)
	}

	var transactions []Transaction
	err = db.Unscoped().Where("import_hash LIKE '[%'").Find(&transactions).Error
	if err != nil {
		return err
	}

	for _, transaction := range transactions {
		// The string looks like this: "[40 52 207 7 118 61 80 107 178 242 5 47 211 161 180 135 104 222 118 28 56 12 33 63 179 78 39 173 206 11 77 3]"
		// With trimming and splitting it, we get a slice containing every individual number
		bytes := strings.Split(strings.TrimRight(strings.TrimLeft(transaction.ImportHash, "["), "]"), " ")

		// Assemble the slice back to a string. We pad with zeroes so that every byte takes two characters
		var b strings.Builder
		for _, part := range bytes {
			// Need to convert to int first so that it's interpreted correctly
			charAsInt, _ := strconv.Atoi(part)
			b.WriteString(fmt.Sprintf("%02x", byte(charAsInt)))
		}

		// Save the record back to the DB
		transaction.ImportHash = b.String()
		db.Unscoped().Save(&transaction)
	}

	return
}
