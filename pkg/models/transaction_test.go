package models_test

import (
	"time"

	"github.com/envelope-zero/backend/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestTransactionFindTimeUTC() {
	tz, _ := time.LoadLocation("Europe/Berlin")

	transaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
		},
	}

	err := transaction.AfterFind(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "transaction.AfterFind failed")
	}

	assert.Equal(suite.T(), time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")
}

func (suite *TestSuiteStandard) TestTransactionSaveTimeUTC() {
	tz, _ := time.LoadLocation("Europe/Berlin")

	transaction := models.Transaction{}
	err := transaction.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "transaction.AfterFind failed")
	}

	assert.Equal(suite.T(), time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")

	transaction = models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
		},
	}
	err = transaction.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "transaction.AfterFind failed")
	}

	assert.Equal(suite.T(), time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")
}
