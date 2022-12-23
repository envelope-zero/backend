package models_test

import (
	"strconv"

	"github.com/envelope-zero/backend/pkg/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestAccountCalculations() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	account := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID:       budget.ID,
			OnBudget:       true,
			External:       false,
			InitialBalance: decimal.NewFromFloat(170),
		},
	}
	err = suite.db.Save(&account).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	externalAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			External: true,
		},
	}
	err = suite.db.Save(&externalAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	category := models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: budget.ID,
		},
	}
	err = suite.db.Save(&category).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	envelope := models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			CategoryID: category.ID,
		},
	}
	err = suite.db.Save(&envelope).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	incomingTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: account.ID,
			Reconciled:           true,
			Amount:               decimal.NewFromFloat(32.17),
		},
	}
	err = suite.db.Save(&incomingTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	outgoingTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      account.ID,
			DestinationAccountID: externalAccount.ID,
			Amount:               decimal.NewFromFloat(17.45),
		},
	}
	err = suite.db.Save(&outgoingTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	a := account.WithCalculations(suite.db)

	assert.True(suite.T(), a.Balance.Equal(incomingTransaction.Amount.Sub(outgoingTransaction.Amount).Add(a.InitialBalance)), "Balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount.Sub(outgoingTransaction.Amount), a.Balance)

	assert.True(suite.T(), a.ReconciledBalance.Equal(incomingTransaction.Amount.Add(a.InitialBalance)), "Reconciled balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount, a.ReconciledBalance)

	err = suite.db.Delete(&incomingTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be deleted", err)
	}

	a = account.WithCalculations(suite.db)
	assert.True(suite.T(), a.Balance.Equal(outgoingTransaction.Amount.Neg().Add(a.InitialBalance)), "Balance for account is not correct. Should be: %v but is %v", outgoingTransaction.Amount.Neg(), a.Balance)
	assert.True(suite.T(), a.ReconciledBalance.Equal(decimal.NewFromFloat(0).Add(a.InitialBalance)), "Reconciled balance for account is not correct. Should be: %v but is %v", decimal.NewFromFloat(0), a.ReconciledBalance)
}

func (suite *TestSuiteStandard) TestAccountTransactions() {
	account := models.Account{}

	transactions := account.Transactions(suite.db)
	assert.Len(suite.T(), transactions, 0)
}

func (suite *TestSuiteStandard) TestAccountOnBudget() {
	account := models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: true,
		},
	}

	err := account.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.False(suite.T(), account.OnBudget, "OnBudget is true even though the account is external")

	account = models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: false,
		},
	}

	err = account.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.True(suite.T(), account.OnBudget, "OnBudget is set to false even though the account is internal")
}

func (suite *TestSuiteStandard) TestAccountRecentEnvelopes() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Budget could not be saved", err)
	}

	account := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID:       budget.ID,
			OnBudget:       true,
			External:       false,
			InitialBalance: decimal.NewFromFloat(170),
		},
	}
	err = suite.db.Save(&account).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	externalAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			External: true,
		},
	}
	err = suite.db.Save(&externalAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	category := models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: budget.ID,
		},
	}
	err = suite.db.Save(&category).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	envelopeIDs := []*uuid.UUID{}
	for i := 1; i <= 3; i++ {
		envelope := &models.Envelope{
			EnvelopeCreate: models.EnvelopeCreate{
				CategoryID: category.ID,
				Name:       strconv.Itoa(i),
			},
		}
		if err = suite.db.Save(&envelope).Error; err != nil {
			suite.Assert().Fail("Resource could not be saved", err)
		}

		envelopeIDs = append(envelopeIDs, &envelope.ID)
	}

	// Create 15 transactions:
	//  * 2 for the first envelope
	//  * 2 for the second envelope
	//  * 11 for the last envelope
	for i := 1; i <= 15; i++ {
		eIndex := i
		if i > 5 {
			eIndex = 2
		}
		transaction := models.Transaction{
			TransactionCreate: models.TransactionCreate{
				BudgetID:             budget.ID,
				EnvelopeID:           envelopeIDs[eIndex%3],
				SourceAccountID:      account.ID,
				DestinationAccountID: externalAccount.ID,
				Amount:               decimal.NewFromFloat(17.45),
			},
		}
		err = suite.db.Save(&transaction).Error
		if err != nil {
			suite.Assert().Fail("Resource could not be saved", err)
		}
	}

	recent, err := externalAccount.RecentEnvelopes(suite.db)
	if err != nil {
		suite.Assert().Fail("Could not compute recent envelopes", err)
	}

	// The last envelope needs to be the first in the sort since it
	// has been the most common one in the last 10 transactions
	suite.Assert().Equal(*envelopeIDs[2], recent[0].ID)

	// The second envelope is as common as the first, but its newest transaction
	// is newer than the first envelope's newest transaction,
	// so it needs to come second
	suite.Assert().Equal(*envelopeIDs[1], recent[1].ID)

	// The first envelope is the last one
	suite.Assert().Equal(*envelopeIDs[0], recent[2].ID)
}
