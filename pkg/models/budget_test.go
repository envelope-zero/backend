package models_test

import (
	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteEnv) TestBudgetCalculations() {
	budget := models.Budget{}
	err := database.DB.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	bankAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			OnBudget: true,
			External: false,
		},
	}
	err = database.DB.Save(&bankAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	cashAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			OnBudget: true,
			External: false,
		},
	}
	err = database.DB.Save(&cashAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	employerAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			External: true,
		},
	}
	err = database.DB.Save(&employerAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	groceryAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			External: true,
		},
	}
	err = database.DB.Save(&groceryAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	category := models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: budget.ID,
		},
	}
	err = database.DB.Save(&category).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	envelope := models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			CategoryID: category.ID,
		},
	}
	err = database.DB.Save(&envelope).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	salaryTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      employerAccount.ID,
			DestinationAccountID: bankAccount.ID,
			Reconciled:           true,
			Amount:               decimal.NewFromFloat(2857.51),
		},
	}
	err = database.DB.Save(&salaryTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	outgoingTransactionBank := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      bankAccount.ID,
			DestinationAccountID: groceryAccount.ID,
			Amount:               decimal.NewFromFloat(87.45),
		},
	}
	err = database.DB.Save(&outgoingTransactionBank).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	outgoingTransactionCash := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      cashAccount.ID,
			DestinationAccountID: groceryAccount.ID,
			Amount:               decimal.NewFromFloat(23.17),
		},
	}
	err = database.DB.Save(&outgoingTransactionCash).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	budget = budget.WithCalculations()

	shouldBalance := decimal.NewFromFloat(2746.89)
	assert.True(suite.T(), budget.Balance.Equal(shouldBalance), "Balance for budget is not correct. Should be %s, is %s", shouldBalance, budget.Balance)
}
