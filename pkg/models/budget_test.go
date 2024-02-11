package models_test

import (
	"strings"
	"time"

	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestBudgetTrimWhitespace() {
	name := "\t Whitespace galore!   "
	note := " Some more whitespace in the notes    "
	currency := "  €"

	budget := suite.createTestBudget(models.Budget{
		Name:     name,
		Note:     note,
		Currency: currency,
	})

	assert.Equal(suite.T(), strings.TrimSpace(name), budget.Name)
	assert.Equal(suite.T(), strings.TrimSpace(note), budget.Note)
	assert.Equal(suite.T(), strings.TrimSpace(currency), budget.Currency)
}

func (suite *TestSuiteStandard) TestBudgetCalculations() {
	// Sum of salary transactions: 7400
	// Sum of income available in March: 4600
	// Sum of all allocations: 91.58
	// Outgoing bank account: 87.45
	// Outgoing cash account: 43.17
	// Outgoing total: 130.62
	// Sum of allocations for Grocery Envelope until 2022-03: 67
	// Allocations for Grocery Envelope - Outgoing transactions = -43.62
	marchTwentyTwentyTwo := types.NewMonth(2022, 3)

	budget := suite.createTestBudget(models.Budget{})
	emptyBudget := suite.createTestBudget(models.Budget{})

	bankAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		OnBudget: true,
		External: false,
		Name:     "TestBudgetCalculations Bank Account",
	})

	cashAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		OnBudget: true,
		External: false,
		Name:     "TestBudgetCalculations Cash Account",
	})

	employerAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		External: true,
		Name:     "TestBudgetCalculations Employer Account",
	})

	groceryAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		External: true,
	})

	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		CategoryID: category.ID,
	})

	_ = suite.createTestMonthConfig(models.MonthConfig{
		EnvelopeID: envelope.ID,
		Month:      marchTwentyTwentyTwo.AddDate(0, -2),
		Allocation: decimal.NewFromFloat(17.42),
	})

	_ = suite.createTestMonthConfig(models.MonthConfig{
		EnvelopeID: envelope.ID,
		Month:      marchTwentyTwentyTwo.AddDate(0, -1),
		Allocation: decimal.NewFromFloat(24.58),
	})

	_ = suite.createTestMonthConfig(models.MonthConfig{
		EnvelopeID: envelope.ID,
		Month:      marchTwentyTwentyTwo,
		Allocation: decimal.NewFromFloat(25),
	})

	_ = suite.createTestMonthConfig(models.MonthConfig{
		EnvelopeID: envelope.ID,
		Month:      types.NewMonth(2170, 2),
		Allocation: decimal.NewFromFloat(24.58),
	})

	_ = suite.createTestTransaction(models.Transaction{
		Date:                 time.Time(marchTwentyTwentyTwo),
		EnvelopeID:           nil,
		SourceAccountID:      employerAccount.ID,
		DestinationAccountID: bankAccount.ID,
		Amount:               decimal.NewFromFloat(1800),
	})

	_ = suite.createTestTransaction(models.Transaction{
		Date:                 time.Time(marchTwentyTwentyTwo),
		EnvelopeID:           nil,
		SourceAccountID:      employerAccount.ID,
		DestinationAccountID: bankAccount.ID,
		Amount:               decimal.NewFromFloat(2800),
	})

	_ = suite.createTestTransaction(models.Transaction{
		Date:                 time.Time(marchTwentyTwentyTwo.AddDate(0, 1)),
		EnvelopeID:           nil,
		SourceAccountID:      employerAccount.ID,
		DestinationAccountID: bankAccount.ID,
		Amount:               decimal.NewFromFloat(2800),
	})

	_ = suite.createTestTransaction(models.Transaction{
		Date:                 time.Time(marchTwentyTwentyTwo),
		EnvelopeID:           &envelope.ID,
		SourceAccountID:      bankAccount.ID,
		DestinationAccountID: groceryAccount.ID,
		Amount:               decimal.NewFromFloat(87.45),
	})

	_ = suite.createTestTransaction(models.Transaction{
		Date:                 time.Time(marchTwentyTwentyTwo),
		EnvelopeID:           &envelope.ID,
		SourceAccountID:      cashAccount.ID,
		DestinationAccountID: groceryAccount.ID,
		Amount:               decimal.NewFromFloat(23.17),
	})

	_ = suite.createTestTransaction(models.Transaction{
		Date:                 time.Time(marchTwentyTwentyTwo),
		SourceAccountID:      cashAccount.ID,
		DestinationAccountID: groceryAccount.ID,
		Amount:               decimal.NewFromFloat(20),
	})

	shouldBalance := decimal.NewFromFloat(7269.38)
	isBalance, err := budget.Balance(models.DB)
	if err != nil {
		assert.FailNow(suite.T(), "Balance for budget could not be calculated")
	}
	assert.True(suite.T(), isBalance.Equal(shouldBalance), "Balance for budget is not correct. Should be %s, is %s", shouldBalance, budget.Balance)

	// Verify income for used budget in March
	shouldIncome := decimal.NewFromFloat(4600)
	income, err := budget.Income(models.DB, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.Equal(shouldIncome), "Income is %s, should be %s", income, shouldIncome)

	// Verify income for empty budget in March
	income, err = emptyBudget.Income(models.DB, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)

	// Verify budgeted for used budget
	budgeted, err := budget.Allocated(models.DB, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.Equal(decimal.NewFromFloat(25)), "Budgeted is %s, should be 25", budgeted)

	// Verify budgeted for empty budget
	budgeted, err = emptyBudget.Allocated(models.DB, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.IsZero(), "Budgeted is %s, should be 0", budgeted)
}

func (suite *TestSuiteStandard) TestMonthIncomeNoTransactions() {
	budget := suite.createTestBudget(models.Budget{})

	income, err := budget.Income(models.DB, types.NewMonth(2022, 3))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)
}

func (suite *TestSuiteStandard) TestBudgetIncomeDBFail() {
	budget := suite.createTestBudget(models.Budget{})

	suite.CloseDB()

	_, err := budget.Income(models.DB, types.NewMonth(1995, 2))
	suite.Assert().ErrorIs(err, models.ErrGeneral)
}

func (suite *TestSuiteStandard) TestBudgetBudgetedDBFail() {
	budget := suite.createTestBudget(models.Budget{})

	suite.CloseDB()

	_, err := budget.Allocated(models.DB, types.NewMonth(200, 2))
	suite.Assert().ErrorIs(err, models.ErrGeneral)
}
