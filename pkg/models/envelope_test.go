package models_test

import (
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteEnv) TestEnvelopeMonthSum() {
	budget := models.Budget{}
	err := database.DB.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	internalAccount := &models.Account{
		AccountCreate: models.AccountCreate{
			Name:     "Internal Source Account",
			BudgetID: budget.ID,
		},
	}
	err = database.DB.Create(internalAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	externalAccount := &models.Account{
		AccountCreate: models.AccountCreate{
			Name:     "External Destination Account",
			BudgetID: budget.ID,
			External: true,
		},
	}
	err = database.DB.Create(&externalAccount).Error
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

	envelope := &models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			Name:       "Testing envelope",
			CategoryID: category.ID,
		},
	}
	err = database.DB.Create(&envelope).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	spent := decimal.NewFromFloat(17.32)
	transaction := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           envelope.ID,
			Amount:               spent,
			SourceAccountID:      internalAccount.ID,
			DestinationAccountID: externalAccount.ID,
			Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	err = database.DB.Create(&transaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	transactionIn := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           envelope.ID,
			Amount:               spent.Neg(),
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: internalAccount.ID,
			Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	err = database.DB.Create(&transactionIn).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	envelopeMonth := envelope.Month(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent.Neg()), "Month calculation for 2022-01 is wrong: should be %v, but is %v", spent.Neg(), envelopeMonth.Spent)

	envelopeMonth = envelope.Month(time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC))
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent.Neg()), "Month calculation for 2022-02 is wrong: should be %v, but is %v", spent, envelopeMonth.Spent)
}
