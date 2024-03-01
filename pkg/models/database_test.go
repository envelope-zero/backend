package models_test

import (
	"os"
	"testing"

	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/envelope-zero/backend/v5/test"
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

	input, err := os.ReadFile("../../testdata/migrations/v4-v5.db")
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
