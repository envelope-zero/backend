package models_test

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/envelope-zero/backend/v7/internal/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Regression test for https://github.com/envelope-zero/backend/issues/1007
	offBudgetAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		OnBudget: false,
		External: false,
		Name:     "TestBudgetCalculations Off Budget AccountAccount",
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

	// Regression test for https://github.com/envelope-zero/backend/issues/1007
	_ = suite.createTestTransaction(models.Transaction{
		Date:                 time.Time(marchTwentyTwentyTwo),
		SourceAccountID:      offBudgetAccount.ID,
		DestinationAccountID: cashAccount.ID,
		Amount:               decimal.NewFromFloat(20),
	})

	shouldBalance := decimal.NewFromFloat(7289.38)
	isBalance, err := budget.Balance(models.DB)
	if err != nil {
		assert.FailNow(suite.T(), "Balance for budget could not be calculated")
	}
	assert.True(suite.T(), isBalance.Equal(shouldBalance), "Balance for budget is not correct. Should be %s, is %s", shouldBalance, isBalance)

	// Verify income for used budget in March. AvailableFrom defaults to next month, so we check for April
	shouldIncome := decimal.NewFromFloat(4620) // Income transaction from employer + income from off budget account
	income, err := budget.Income(models.DB, marchTwentyTwentyTwo.AddDate(0, 1))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.Equal(shouldIncome), "Income is %s, should be %s", income, shouldIncome)

	// Verify income for empty budget in March. AvailableFrom defaults to next month, so we check for April
	income, err = emptyBudget.Income(models.DB, marchTwentyTwentyTwo.AddDate(0, 1))
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

func (suite *TestSuiteStandard) TestBudgetExport() {
	t := suite.T()

	_ = suite.createTestBudget(models.Budget{
		Name: "TestBudgetExport",
	})

	raw, err := models.Budget{}.Export()
	if err != nil {
		require.Fail(t, "budget export failed", err)
	}

	var budgets []models.Budget
	err = json.Unmarshal(raw, &budgets)
	if err != nil {
		require.Fail(t, "JSON could not be unmarshaled", err)
	}

	require.Len(t, budgets, 1, "Number of budgets in export is wrong")
}
