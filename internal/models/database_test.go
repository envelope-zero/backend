package models_test

import (
	"os"
	"testing"

	"github.com/envelope-zero/backend/v5/internal/models"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateWithExistingDB(t *testing.T) {
	testDB := test.TmpFile(t)

	// Migrate the database once
	require.Nil(t, models.Connect(testDB))

	// Close the connection
	sqlDB, err := models.DB.DB()
	require.Nil(t, err)
	sqlDB.Close()

	// Migrate it again
	require.Nil(t, models.Connect(testDB))
}

// TestV4V5Migration tests the migration from v4 to v5
func TestV4V5Migration(t *testing.T) {
	dbFile := test.TmpFile(t)

	input, err := os.ReadFile("../../test/data/migrations/v4-v5.db")
	if err != nil {
		t.Error("Could not read test database")
	}
	err = os.WriteFile(dbFile, input, 0o644)
	if err != nil {
		t.Error("Could not create temporary copy for database")
	}

	// Connect to the database
	require.Nil(t, models.Connect(dbFile))

	// Close the connection
	sqlDB, err := models.DB.DB()
	require.Nil(t, err)
	sqlDB.Close()

	// Reconnect
	require.Nil(t, models.Connect(dbFile))
}

// TestSoftDeleteMigration tests the removal of the DeletedAt field
func TestSoftDeleteMigration(t *testing.T) {
	dbFile := test.TmpFile(t)

	input, err := os.ReadFile("../../test/data/migrations/remove-soft-delete.db")
	if err != nil {
		t.Error("Could not read test database")
	}
	err = os.WriteFile(dbFile, input, 0o644)
	if err != nil {
		t.Error("Could not create temporary copy for database")
	}

	// Connect to the database
	require.Nil(t, models.Connect(dbFile))

	// Close the connection
	sqlDB, err := models.DB.DB()
	require.Nil(t, err)
	sqlDB.Close()

	// Reconnect
	require.Nil(t, models.Connect(dbFile))

	// Verify that all deleted resources have been deleted
	var accounts []models.Account
	models.DB.Model(&models.Account{}).Find(&accounts)
	assert.Len(t, accounts, 0, "Soft-deleted accounts have not been deleted during the migration")

	var budgets []models.Budget
	models.DB.Model(&models.Budget{}).Find(&budgets)
	assert.Len(t, budgets, 0, "Soft-deleted budgets have not been deleted during the migration")

	var categories []models.Category
	models.DB.Model(&models.Category{}).Find(&categories)
	assert.Len(t, categories, 0, "Soft-deleted categories have not been deleted during the migration")

	var envelopes []models.Envelope
	models.DB.Model(&models.Envelope{}).Find(&envelopes)
	assert.Len(t, envelopes, 0, "Soft-deleted envelopes have not been deleted during the migration")

	var goals []models.Goal
	models.DB.Model(&models.Goal{}).Find(&goals)
	assert.Len(t, goals, 0, "Soft-deleted goals have not been deleted during the migration")

	var matchRules []models.MatchRule
	models.DB.Model(&models.MatchRule{}).Find(&matchRules)
	assert.Len(t, matchRules, 0, "Soft-deleted match rules have not been deleted during the migration")

	var transactions []models.Transaction
	models.DB.Model(&models.Transaction{}).Find(&transactions)
	assert.Len(t, transactions, 0, "Soft-deleted transactions have not been deleted during the migration")
}
