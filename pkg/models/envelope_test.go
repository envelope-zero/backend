package models_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestEnvelopeTrimWhitespace() {
	name := "\t Whitespace galore!   "
	note := " Some more whitespace in the notes    "

	envelope := suite.createTestEnvelope(models.Envelope{
		Name: name,
		Note: note,
		CategoryID: suite.createTestCategory(models.Category{
			BudgetID: suite.createTestBudget(models.Budget{}).ID,
		}).ID,
	})

	assert.Equal(suite.T(), strings.TrimSpace(name), envelope.Name)
	assert.Equal(suite.T(), strings.TrimSpace(note), envelope.Note)
}

func (suite *TestSuiteStandard) TestEnvelopeMonthSum() {
	budget := suite.createTestBudget(models.Budget{})

	internalAccount := suite.createTestAccount(models.Account{
		Name:     "Internal Source Account",
		BudgetID: budget.ID,
		OnBudget: true,
	})

	externalAccount := suite.createTestAccount(models.Account{
		Name:     "External Destination Account",
		BudgetID: budget.ID,
		External: true,
	})

	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		Name:       "Testing envelope",
		CategoryID: category.ID,
	})

	january := types.NewMonth(2022, 1)

	spent := decimal.NewFromFloat(17.32)
	transaction := suite.createTestTransaction(models.Transaction{
		BudgetID:             budget.ID,
		EnvelopeID:           &envelope.ID,
		Amount:               spent,
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 time.Time(january),
	})

	_ = suite.createTestTransaction(models.Transaction{
		BudgetID:             budget.ID,
		EnvelopeID:           &envelope.ID,
		Amount:               spent,
		SourceAccountID:      externalAccount.ID,
		DestinationAccountID: internalAccount.ID,
		Date:                 time.Time(january.AddDate(0, 1)),
	})

	envelopeMonth, err := envelope.Month(models.DB, january)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent.Neg()), "Month calculation for 2022-01 is wrong: should be %v, but is %v", spent.Neg(), envelopeMonth.Spent)

	envelopeMonth, err = envelope.Month(models.DB, january.AddDate(0, 1))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(spent), "Month calculation for 2022-02 is wrong: should be %v, but is %v", spent, envelopeMonth.Spent)

	err = models.DB.Delete(&transaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be deleted", err)
	}

	envelopeMonth, err = envelope.Month(models.DB, january)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelopeMonth.Spent.Equal(decimal.NewFromFloat(0)), "Month calculation for 2022-01 is wrong: should be %v, but is %v", decimal.NewFromFloat(0), envelopeMonth.Spent)
}

func (suite *TestSuiteStandard) TestCreateTransactionNoEnvelope() {
	budget := suite.createTestBudget(models.Budget{})

	internalAccount := suite.createTestAccount(models.Account{
		Name:     "Internal Source Account",
		BudgetID: budget.ID,
	})

	externalAccount := suite.createTestAccount(models.Account{
		Name:     "External Destination Account",
		BudgetID: budget.ID,
		External: true,
	})

	_ = suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	// Transactions must be able to be created without an envelope (to enable internal transfers without an Envelope and income transactions)
	_ = suite.createTestTransaction(models.Transaction{
		BudgetID:             budget.ID,
		Amount:               decimal.NewFromFloat(17.32),
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
	})
}

// TestEnvelopeMonthBalance verifies that the monthly balance calculation for
// envelopes is correct.
func (suite *TestSuiteStandard) TestEnvelopeMonthBalance() {
	budget := suite.createTestBudget(models.Budget{})

	internalAccount := suite.createTestAccount(models.Account{
		Name:     "Internal Source Account",
		BudgetID: budget.ID,
		OnBudget: true,
	})

	externalAccount := suite.createTestAccount(models.Account{
		Name:     "External Destination Account",
		BudgetID: budget.ID,
		External: true,
	})

	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		Name:       "Testing envelope",
		CategoryID: category.ID,
	})

	// Used to test the Envelope.Balance method without any transactions
	envelopeWithoutTransactions := suite.createTestEnvelope(models.Envelope{
		Name:       "Testing envelope without any transactions",
		CategoryID: category.ID,
	})

	january := types.NewMonth(2022, 1)

	// Allocation in January
	_ = suite.createTestMonthConfig(models.MonthConfig{
		EnvelopeID: envelope.ID,
		Month:      january,
		Allocation: decimal.NewFromFloat(50),
	})

	// Allocation in February
	_ = suite.createTestMonthConfig(models.MonthConfig{
		EnvelopeID: envelope.ID,
		Month:      january.AddDate(0, 1),
		Allocation: decimal.NewFromFloat(40),
	})

	// Transaction in January
	_ = suite.createTestTransaction(models.Transaction{
		BudgetID:             budget.ID,
		EnvelopeID:           &envelope.ID,
		Amount:               decimal.NewFromFloat(15),
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 time.Time(january),
	})

	// Transaction in February
	_ = suite.createTestTransaction(models.Transaction{
		BudgetID:             budget.ID,
		EnvelopeID:           &envelope.ID,
		Amount:               decimal.NewFromFloat(30),
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 time.Time(january.AddDate(0, 1)),
	})

	// Deleted transaction to verify that deleted transactions are not used in the calculation
	deletedTransaction := suite.createTestTransaction(models.Transaction{
		BudgetID:             budget.ID,
		EnvelopeID:           &envelope.ID,
		Amount:               decimal.NewFromFloat(30),
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 time.Time(january.AddDate(0, 1)),
	})
	models.DB.Delete(&deletedTransaction)

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
			eMonth, err := tt.envelope.Month(models.DB, tt.month)
			assert.Nil(t, err)
			assert.True(t, eMonth.Balance.Equal(should), "Balance calculation for 2022-01 is wrong: should be %v, but is %v", should, eMonth.Balance)
		})
	}
}

// TestEnvelopeUnarchiveUnarchivesCategory tests that when an envelope is unarchived, but its parent category
// is archived, the parent category is unarchived, too.
func (suite *TestSuiteStandard) TestEnvelopeUnarchiveUnarchivesCategory() {
	budget := suite.createTestBudget(models.Budget{})
	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
		Archived: true,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		CategoryID: category.ID,
		Name:       "TestEnvelopeUnarchiveUnarchivesCategory",
		Archived:   true,
	})

	// Unarchive the envelope
	data := models.Envelope{Archived: false}
	models.DB.Model(&envelope).Select("Archived").Updates(data)

	// Reload the category
	models.DB.First(&category, category.ID)
	assert.False(suite.T(), category.Archived, "Category should be unarchived when child envelope is unarchived")
}

func (suite *TestSuiteStandard) TestEnvelopeSelf() {
	assert.Equal(suite.T(), "Envelope", models.Envelope{}.Self())
}
