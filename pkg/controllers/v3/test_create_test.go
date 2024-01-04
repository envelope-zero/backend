package v3_test

import (
	v3 "github.com/envelope-zero/backend/v4/pkg/controllers/v3"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/google/uuid"
)

func (suite *TestSuiteStandard) defaultTransactionCreate(c models.Transaction) models.Transaction {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = suite.createTestBudget(suite.T(), v3.BudgetEditable{Name: "Testing budget"}).Data.ID
	}

	if c.SourceAccountID == uuid.Nil {
		c.SourceAccountID = suite.createTestAccount(suite.T(), models.Account{Name: "Source Account"}).Data.ID
	}

	if c.DestinationAccountID == uuid.Nil {
		c.DestinationAccountID = suite.createTestAccount(suite.T(), models.Account{Name: "Destination Account"}).Data.ID
	}

	if c.EnvelopeID == &uuid.Nil {
		*c.EnvelopeID = suite.createTestEnvelope(suite.T(), v3.EnvelopeCreate{Name: "Transaction Test Envelope"}).Data.ID
	}

	return c
}
