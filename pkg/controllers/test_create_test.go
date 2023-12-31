package controllers_test

import (
	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/google/uuid"
)

func (suite *TestSuiteStandard) defaultTransactionCreate(c models.TransactionCreate) models.TransactionCreate {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = suite.createTestBudgetV3(suite.T(), models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	if c.SourceAccountID == uuid.Nil {
		c.SourceAccountID = suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "Source Account"}).Data.ID
	}

	if c.DestinationAccountID == uuid.Nil {
		c.DestinationAccountID = suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "Destination Account"}).Data.ID
	}

	if c.EnvelopeID == &uuid.Nil {
		*c.EnvelopeID = suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{Name: "Transaction Test Envelope"}).Data.ID
	}

	return c
}
