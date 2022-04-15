package controllers_test

import (
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/controllers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDBConnectionErrorHandled(t *testing.T) {
	os.Setenv("DB_HOST", "invalid")

	_, err := controllers.Router()

	assert.NotNil(t, err)
	os.Unsetenv("DB_HOST")
}

func TestGinMode(t *testing.T) {
	os.Setenv("GIN_MODE", "debug")
	_, err := controllers.Router()

	assert.Nil(t, err, "%T: %v", err, err)
	assert.True(t, gin.IsDebugging())

	os.Unsetenv("GIN_MODE")
}
