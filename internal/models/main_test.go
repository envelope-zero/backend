package models_test

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMain takes care of the test setup for this package.
func TestMain(m *testing.M) {
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode("release")
	}

	m.Run()
}
