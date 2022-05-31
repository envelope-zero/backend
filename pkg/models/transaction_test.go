package models_test

import (
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestTransactionFindTimeUTC(t *testing.T) {
	tz, _ := time.LoadLocation("Europe/Berlin")

	transaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
		},
	}

	err := transaction.AfterFind(database.DB)
	if err != nil {
		assert.Fail(t, "transaction.AfterFind failed")
	}

	assert.Equal(t, time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")
}

func TestTransactionSaveTimeUTC(t *testing.T) {
	tz, _ := time.LoadLocation("Europe/Berlin")

	transaction := models.Transaction{}
	err := transaction.BeforeSave(database.DB)
	if err != nil {
		assert.Fail(t, "transaction.AfterFind failed")
	}

	assert.Equal(t, time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")

	transaction = models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
		},
	}
	err = transaction.BeforeSave(database.DB)
	if err != nil {
		assert.Fail(t, "transaction.AfterFind failed")
	}

	assert.Equal(t, time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")
}
