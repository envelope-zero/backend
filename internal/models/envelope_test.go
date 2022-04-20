package models_test

import (
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestEnvelopeMonthSum(t *testing.T) {
	internalAccount := &models.Account{
		Name: "Internal Source Account",
	}
	models.DB.Create(internalAccount)

	externalAccount := &models.Account{
		Name:     "External Destination Account",
		External: true,
	}
	models.DB.Create(&externalAccount)

	envelope := &models.Envelope{
		Name: "Testing envelope",
	}
	models.DB.Create(&envelope)

	spent := decimal.NewFromFloat(17.32)
	transaction := &models.Transaction{
		EnvelopeID:           envelope.ID,
		Amount:               spent,
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	models.DB.Create(&transaction)

	transactionIn := &models.Transaction{
		EnvelopeID:           envelope.ID,
		Amount:               spent.Neg(),
		SourceAccountID:      externalAccount.ID,
		DestinationAccountID: internalAccount.ID,
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
	}
	models.DB.Create(&transactionIn)

	envelopeMonth := envelope.Month(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.True(t, envelopeMonth.Spent.Equal(spent.Neg()), "Month calculation for 2022-01 is wrong: should be %v, but is %v", spent.Neg(), envelopeMonth.Spent)

	envelopeMonth = envelope.Month(time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC))
	assert.True(t, envelopeMonth.Spent.Equal(spent.Neg()), "Month calculation for 2022-02 is wrong: should be %v, but is %v", spent, envelopeMonth.Spent)
}
