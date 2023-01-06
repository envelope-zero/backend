package models_test

import (
	"time"

	"github.com/envelope-zero/backend/internal/types"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestEnvelopeMonthSum() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	internalAccount := &models.Account{
		AccountCreate: models.AccountCreate{
			Name:     "Internal Source Account",
			BudgetID: budget.ID,
			OnBudget: true,
		},
	}
	err = suite.db.Create(internalAccount).Error
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
	err = suite.db.Create(&externalAccount).Error
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

	envelope := &models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			Name:       "Testing envelope",
			CategoryID: category.ID,
		},
	}
	err = suite.db.Create(&envelope).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	january := types.NewMonth(2022, 1)

	spent := decimal.NewFromFloat(17.32)
	transaction := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			Amount:               spent,
			SourceAccountID:      internalAccount.ID,
			DestinationAccountID: externalAccount.ID,
			Date:                 time.Time(january),
		},
	}
	err = suite.db.Create(&transaction).Error
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
			Date:                 time.Time(january.AddDate(0, 1)),
		},
	}
	err = suite.db.Create(&transactionIn).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	envelopeMonth, _, err := envelope.Month(suite.db, january)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent), "Month calculation for 2022-01 is wrong: should be %v, but is %v", spent, envelopeMonth.Spent)

	envelopeMonth, _, err = envelope.Month(suite.db, january.AddDate(0, 1))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent), "Month calculation for 2022-02 is wrong: should be %v, but is %v", spent, envelopeMonth.Spent)

	err = suite.db.Delete(&transaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be deleted", err)
	}

	envelopeMonth, _, err = envelope.Month(suite.db, january)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(decimal.NewFromFloat(0)), "Month calculation for 2022-01 is wrong: should be %v, but is %v", decimal.NewFromFloat(0), envelopeMonth.Spent)
}

func (suite *TestSuiteStandard) TestCreateTransactionNoEnvelope() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	internalAccount := &models.Account{
		AccountCreate: models.AccountCreate{
			Name:     "Internal Source Account",
			BudgetID: budget.ID,
		},
	}
	err = suite.db.Create(internalAccount).Error
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
	err = suite.db.Create(&externalAccount).Error
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

	transaction := &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			Amount:               decimal.NewFromFloat(17.32),
			SourceAccountID:      internalAccount.ID,
			DestinationAccountID: externalAccount.ID,
			Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	err = suite.db.Create(&transaction).Error

	assert.Nil(suite.T(), err, "Transactions must be able to be created without an envelope (to enable internal transfers without an Envelope and income transactions)")
}

func (suite *TestSuiteStandard) TestEnvelopeMonthBalance() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	internalAccount := &models.Account{
		AccountCreate: models.AccountCreate{
			Name:     "Internal Source Account",
			BudgetID: budget.ID,
		},
	}
	err = suite.db.Create(internalAccount).Error
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
	err = suite.db.Create(&externalAccount).Error
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

	envelope := &models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			Name:       "Testing envelope",
			CategoryID: category.ID,
		},
	}
	err = suite.db.Create(&envelope).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	// Used to test the Envelope.Balance method without any transactions
	envelope2 := &models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			Name:       "Testing envelope without any transactions",
			CategoryID: category.ID,
		},
	}
	err = suite.db.Create(&envelope2).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	january := types.NewMonth(2022, 1)

	allocationJan := &models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Month:      january,
			Amount:     decimal.NewFromFloat(50),
		},
	}
	err = suite.db.Create(&allocationJan).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	allocationFeb := &models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Month:      january.AddDate(0, 1),
			Amount:     decimal.NewFromFloat(40),
		},
	}
	err = suite.db.Create(&allocationFeb).Error
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
			Date:                 time.Time(january),
		},
	}
	err = suite.db.Create(&transaction).Error
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
			Date:                 time.Time(january.AddDate(0, 1)),
		},
	}
	err = suite.db.Create(&transaction2).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	shouldBalance := decimal.NewFromFloat(35)
	envelopeMonth, _, err := envelope.Month(suite.db, january)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Balance.Equal(shouldBalance), "Balance calculation for 2022-01 is wrong: should be %v, but is %v", shouldBalance, envelopeMonth.Balance)

	shouldBalance = decimal.NewFromFloat(45)
	envelopeMonth, _, err = envelope.Month(suite.db, january.AddDate(0, 1))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Balance.Equal(shouldBalance), "Balance calculation for 2022-02 is wrong: should be %v, but is %v", shouldBalance, envelopeMonth.Balance)

	// Verify balance for December (regression test for using AddDate(0, 1, 0) with the month instead of the whole date)
	shouldBalance = decimal.NewFromFloat(45)
	envelopeMonth, _, err = envelope.Month(suite.db, types.NewMonth(2022, 12))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Balance.Equal(shouldBalance), "Balance calculation for 2022-02 is wrong: should be %v, but is %v", shouldBalance, envelopeMonth.Balance)

	shouldBalance = decimal.NewFromFloat(0)
	envelopeMonth, _, err = envelope2.Month(suite.db, january.AddDate(0, 1))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Balance.Equal(shouldBalance), "Balance calculation for 2022-02 is wrong: should be %v, but is %v", shouldBalance, envelopeMonth.Balance)
}
