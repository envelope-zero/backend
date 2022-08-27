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

	january := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	spent := decimal.NewFromFloat(17.32)
	transaction := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			Amount:               spent,
			SourceAccountID:      internalAccount.ID,
			DestinationAccountID: externalAccount.ID,
			Date:                 january,
		},
	}
	err = database.DB.Create(&transaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	transactionIn := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			Amount:               spent.Neg(),
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: internalAccount.ID,
			Date:                 january.AddDate(0, 1, 0),
		},
	}
	err = database.DB.Create(&transactionIn).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	envelopeMonth, err := envelope.Month(january)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent.Neg()), "Month calculation for 2022-01 is wrong: should be %v, but is %v", spent.Neg(), envelopeMonth.Spent)

	envelopeMonth, err = envelope.Month(january.AddDate(0, 1, 0))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent.Neg()), "Month calculation for 2022-02 is wrong: should be %v, but is %v", spent, envelopeMonth.Spent)

	err = database.DB.Delete(&transaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be deleted", err)
	}

	envelopeMonth, err = envelope.Month(january)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(decimal.NewFromFloat(0)), "Month calculation for 2022-01 is wrong: should be %v, but is %v", decimal.NewFromFloat(0), envelopeMonth.Spent)
}

func (suite *TestSuiteEnv) TestCreateTransactionNoEnvelope() {
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

	transaction := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			Amount:               decimal.NewFromFloat(17.32),
			SourceAccountID:      internalAccount.ID,
			DestinationAccountID: externalAccount.ID,
			Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	err = database.DB.Create(&transaction).Error

	assert.Nil(suite.T(), err, "Transactions must be able to be created without an envelope (to enable internal transfers without an Envelope and income transactions)")
}

func (suite *TestSuiteEnv) TestEnvelopeMonthBalance() {
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

	january := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	allocationJan := &models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Month:      january,
			Amount:     decimal.NewFromFloat(50),
		},
	}
	err = database.DB.Create(&allocationJan).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	allocationFeb := &models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Month:      january.AddDate(0, 1, 0),
			Amount:     decimal.NewFromFloat(40),
		},
	}
	err = database.DB.Create(&allocationFeb).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	transaction := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			Amount:               decimal.NewFromFloat(15),
			SourceAccountID:      internalAccount.ID,
			DestinationAccountID: externalAccount.ID,
			Date:                 january,
		},
	}
	err = database.DB.Create(&transaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	transaction2 := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			Amount:               decimal.NewFromFloat(30),
			SourceAccountID:      internalAccount.ID,
			DestinationAccountID: externalAccount.ID,
			Date:                 january.AddDate(0, 1, 0),
		},
	}
	err = database.DB.Create(&transaction2).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	shouldBalance := decimal.NewFromFloat(35)
	envelopeMonth, err := envelope.Month(january)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Balance.Equal(shouldBalance), "Balance calculation for 2022-01 is wrong: should be %v, but is %v", shouldBalance, envelopeMonth.Balance)

	shouldBalance = decimal.NewFromFloat(45)
	envelopeMonth, err = envelope.Month(january.AddDate(0, 1, 0))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Balance.Equal(shouldBalance), "Balance calculation for 2022-02 is wrong: should be %v, but is %v", shouldBalance, envelopeMonth.Balance)
}
