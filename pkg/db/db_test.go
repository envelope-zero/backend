package db_test

import (
	"testing"

	"github.com/envelope-zero/backend/v7/pkg/db"
	"github.com/envelope-zero/backend/v7/test"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	testDB := test.TmpFile(t)

	// Test database connection
	require.Nil(t, db.Connect(testDB))
}
