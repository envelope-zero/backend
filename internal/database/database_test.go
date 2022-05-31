package database_test

import (
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/router"
	"github.com/stretchr/testify/assert"
)

func TestDBConnectionErrorHandled(t *testing.T) {
	os.Setenv("DB_HOST", "invalid")

	_, err := router.Router()

	assert.NotNil(t, err)
	os.Unsetenv("DB_HOST")
}

func TestDBConnection(t *testing.T) {
	_, err := router.Router()

	assert.Nil(t, err)
}
