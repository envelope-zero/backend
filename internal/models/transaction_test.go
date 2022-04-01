package models_test

import (
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestTransactionFindTimeUTC(t *testing.T) {
	tz, _ := time.LoadLocation("Europe/Berlin")

	transaction := models.Transaction{
		Date: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
	}

	err := transaction.AfterFind(models.DB)
	if err != nil {
		assert.Fail(t, "transaction.AfterFind failed")
	}

	assert.Equal(t, time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")
}

func TestTransactionSaveTimeUTC(t *testing.T) {
	tz, _ := time.LoadLocation("Europe/Berlin")

	transaction := models.Transaction{}
	err := transaction.BeforeSave(models.DB)
	if err != nil {
		assert.Fail(t, "transaction.AfterFind failed")
	}

	assert.Equal(t, time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")

	transaction = models.Transaction{
		Date: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
	}
	err = transaction.BeforeSave(models.DB)
	if err != nil {
		assert.Fail(t, "transaction.AfterFind failed")
	}

	assert.Equal(t, time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")
}
