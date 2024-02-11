package models_test

import (
	"fmt"
	"testing"

	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/stretchr/testify/require"
)

func TestMigrateWithExistingDB(t *testing.T) {
	testDB := test.TmpFile(t)

	// Migrate the database once
	require.Nil(t, models.Connect(fmt.Sprintf("%s?_pragma=foreign_keys(1)", testDB)))

	// Close the connection
	sqlDB, err := models.DB.DB()
	require.Nil(t, err)
	sqlDB.Close()

	// Migrate it again
	require.Nil(t, models.Connect(fmt.Sprintf("%s?_pragma=foreign_keys(1)", testDB)))
}
