package db_test

import (
	"testing"

	"github.com/envelope-zero/backend/v5/pkg/db"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	testDB := test.TmpFile(t)

	// Test database connection
	require.Nil(t, db.Connect(testDB))
}
