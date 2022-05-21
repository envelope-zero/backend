package models_test

import (
	"testing"

	"github.com/envelope-zero/backend/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestRawTransactions(t *testing.T) {
	_, err := models.RawTransactions("INVALID query string")

	assert.NotNil(t, err)
}
