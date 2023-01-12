package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v2/internal/types"
	"github.com/envelope-zero/backend/v2/pkg/models"
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
			Amount:               spent,
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
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent.Neg()), "Month calculation for 2022-01 is wrong: should be %v, but is %v", spent.Neg(), envelopeMonth.Spent)

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

// TestEnvelopeMonthBalance verifies that the monthly balance calculation for
// envelopes is correct.
func (suite *TestSuiteStandard) TestEnvelopeMonthBalance() {
	budget := suite.createTestBudget(models.BudgetCreate{})

	internalAccount := suite.createTestAccount(models.AccountCreate{
		Name:     "Internal Source Account",
		BudgetID: budget.ID,
	})

	externalAccount := suite.createTestAccount(models.AccountCreate{
		Name:     "External Destination Account",
		BudgetID: budget.ID,
		External: true,
	})

	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		Name:       "Testing envelope",
		CategoryID: category.ID,
	})

	// Used to test the Envelope.Balance method without any transactions
	envelopeWithoutTransactions := suite.createTestEnvelope(models.EnvelopeCreate{
		Name:       "Testing envelope without any transactions",
		CategoryID: category.ID,
	})

	january := types.NewMonth(2022, 1)

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.ID,
		Month:      january,
		Amount:     decimal.NewFromFloat(50),
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.ID,
		Month:      january.AddDate(0, 1),
		Amount:     decimal.NewFromFloat(40),
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		BudgetID:             budget.ID,
		EnvelopeID:           &envelope.ID,
		Amount:               decimal.NewFromFloat(15),
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 time.Time(january),
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		BudgetID:             budget.ID,
		EnvelopeID:           &envelope.ID,
		Amount:               decimal.NewFromFloat(30),
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 time.Time(january.AddDate(0, 1)),
	})

	tests := []struct {
		month    types.Month
		envelope models.Envelope
		balance  float32
	}{
		{january, envelope, 35},
		{january.AddDate(0, 1), envelope, 45},
		{types.NewMonth(2022, 12), envelope, 45}, // Verify balance for December (regression test for using AddDate(0, 1, 0) with the month instead of the whole date)
		{january.AddDate(0, 1), envelopeWithoutTransactions, 0},
	}

	for _, tt := range tests {
		suite.T().Run(fmt.Sprintf("%s-%s", tt.envelope.Name, tt.month.String()), func(t *testing.T) {
			should := decimal.NewFromFloat(float64(tt.balance))
			eMonth, _, err := tt.envelope.Month(suite.db, tt.month)
			assert.Nil(t, err)
			assert.True(t, eMonth.Balance.Equal(should), "Balance calculation for 2022-01 is wrong: should be %v, but is %v", should, eMonth.Balance)
		})
	}
}
