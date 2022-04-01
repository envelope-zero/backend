package models_test

import (
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/controllers"
	"github.com/stretchr/testify/assert"
)

func TestDBConnectionErrorHandled(t *testing.T) {
	os.Setenv("DB_HOST", "invalid")

	_, err := controllers.Router()

	assert.NotNil(t, err)
	os.Unsetenv("DB_HOST")
}
