package controllers_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/google/uuid"
)

func (suite *TestSuiteStandard) defaultTransactionCreate(c models.TransactionCreate) models.TransactionCreate {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = suite.createTestBudget(models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	if c.SourceAccountID == uuid.Nil {
		c.SourceAccountID = suite.createTestAccount(models.AccountCreate{Name: "Source Account"}).Data.ID
	}

	if c.DestinationAccountID == uuid.Nil {
		c.DestinationAccountID = suite.createTestAccount(models.AccountCreate{Name: "Destination Account"}).Data.ID
	}

	if c.EnvelopeID == &uuid.Nil {
		*c.EnvelopeID = suite.createTestEnvelope(models.EnvelopeCreate{Name: "Transaction Test Envelope"}).Data.ID
	}

	return c
}
