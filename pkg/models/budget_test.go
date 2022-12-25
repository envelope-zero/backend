package models_test

import (
	"time"

	"github.com/envelope-zero/backend/internal/types"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

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

	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	emptyBudget := models.Budget{}
	err = suite.db.Save(&emptyBudget).Error
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
	err = suite.db.Save(&bankAccount).Error
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
	err = suite.db.Save(&cashAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	employerAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			External: true,
		},
	}
	err = suite.db.Save(&employerAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	groceryAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			External: true,
		},
	}
	err = suite.db.Save(&groceryAccount).Error
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

	allocation1 := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Amount:     decimal.NewFromFloat(17.42),
			Month:      marchTwentyTwentyTwo.AddDate(0, -2),
		},
	}
	err = suite.db.Save(&allocation1).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	allocation2 := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Amount:     decimal.NewFromFloat(24.58),
			Month:      marchTwentyTwentyTwo.AddDate(0, -1),
		},
	}
	err = suite.db.Save(&allocation2).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	allocationCurrentMonth := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Amount:     decimal.NewFromFloat(25),
			Month:      marchTwentyTwentyTwo,
		},
	}
	err = suite.db.Save(&allocationCurrentMonth).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	allocationFuture := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Amount:     decimal.NewFromFloat(24.58),
			Month:      types.NewMonth(2170, 2),
		},
	}
	err = suite.db.Save(&allocationFuture).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	salaryTransactionFebruary := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Time(marchTwentyTwentyTwo),
			BudgetID:             budget.ID,
			EnvelopeID:           nil,
			SourceAccountID:      employerAccount.ID,
			DestinationAccountID: bankAccount.ID,
			Reconciled:           true,
			Amount:               decimal.NewFromFloat(1800),
		},
	}
	err = suite.db.Save(&salaryTransactionFebruary).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	salaryTransactionMarch := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Time(marchTwentyTwentyTwo),
			BudgetID:             budget.ID,
			EnvelopeID:           nil,
			SourceAccountID:      employerAccount.ID,
			DestinationAccountID: bankAccount.ID,
			Reconciled:           true,
			Amount:               decimal.NewFromFloat(2800),
		},
	}
	err = suite.db.Save(&salaryTransactionMarch).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	salaryTransactionApril := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Time(marchTwentyTwentyTwo.AddDate(0, 1)),
			BudgetID:             budget.ID,
			EnvelopeID:           nil,
			SourceAccountID:      employerAccount.ID,
			DestinationAccountID: bankAccount.ID,
			Reconciled:           true,
			Amount:               decimal.NewFromFloat(2800),
		},
	}
	err = suite.db.Save(&salaryTransactionApril).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	outgoingTransactionBank := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Time(marchTwentyTwentyTwo),
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      bankAccount.ID,
			DestinationAccountID: groceryAccount.ID,
			Amount:               decimal.NewFromFloat(87.45),
		},
	}
	err = suite.db.Save(&outgoingTransactionBank).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	outgoingTransactionCash := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Time(marchTwentyTwentyTwo),
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      cashAccount.ID,
			DestinationAccountID: groceryAccount.ID,
			Amount:               decimal.NewFromFloat(23.17),
		},
	}
	err = suite.db.Save(&outgoingTransactionCash).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	overspendTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Time(marchTwentyTwentyTwo),
			BudgetID:             budget.ID,
			SourceAccountID:      cashAccount.ID,
			DestinationAccountID: groceryAccount.ID,
			Amount:               decimal.NewFromFloat(20),
		},
	}
	err = suite.db.Save(&overspendTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	budget = budget.WithCalculations(suite.db)

	shouldBalance := decimal.NewFromFloat(7269.38)
	assert.True(suite.T(), budget.Balance.Equal(shouldBalance), "Balance for budget is not correct. Should be %s, is %s", shouldBalance, budget.Balance)

	// Verify income for used budget in March
	shouldIncome := decimal.NewFromFloat(4600)
	income, err := budget.Income(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.Equal(shouldIncome), "Income is %s, should be %s", income, shouldIncome)

	// Verify total income for used budget
	shouldIncomeTotal := decimal.NewFromFloat(4600)
	income, err = budget.TotalIncome(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.Equal(shouldIncomeTotal), "Income is %s, should be %s", income, shouldIncomeTotal)

	// Verify income for empty budget in March
	income, err = emptyBudget.Income(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)

	// Verify total income for empty budget
	income, err = emptyBudget.TotalIncome(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)

	// Verify total budgeted for used budget
	budgeted, err := budget.TotalBudgeted(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.Equal(decimal.NewFromFloat(67)), "Budgeted is %s, should be 67", budgeted)

	// Verify total budgeted for used budget in January after (regression test for using AddDate(0, 1, 0) with the month instead of the whole date)
	budgeted, err = budget.TotalBudgeted(suite.db, types.NewMonth(2023, 1))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.Equal(decimal.NewFromFloat(67)), "Budgeted is %s, should be 67", budgeted)

	// Verify budgeted for used budget
	budgeted, err = budget.Budgeted(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.Equal(decimal.NewFromFloat(25)), "Budgeted is %s, should be 25", budgeted)

	// Verify total budgeted for empty budget
	budgeted, err = emptyBudget.TotalBudgeted(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.IsZero(), "Budgeted is %s, should be 0", budgeted)

	// Verify budgeted for empty budget
	budgeted, err = emptyBudget.Budgeted(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.IsZero(), "Budgeted is %s, should be 0", budgeted)

	// Verify overspent calculation for month without spend
	overspent, err := budget.Overspent(suite.db, types.NewMonth(2022, 1))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), overspent.IsZero(), "Overspent is %s, should be 0", overspent)

	// Verify overspent calculation for month with spend
	overspent, err = budget.Overspent(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), overspent.Equal(decimal.NewFromFloat(63.62)), "Overspent is %s, should be 63.62", overspent)

	// Verify available in month with overspend
	available, err := budget.Available(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), available.Equal(decimal.NewFromFloat(4533)), "Available is %s, should be 4533", available)

	// Verify available in month after overspend
	available, err = budget.Available(suite.db, marchTwentyTwentyTwo.AddDate(0, 1))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), available.Equal(decimal.NewFromFloat(7269.38)), "Available is %s, should be 7269.38", available)
}

func (suite *TestSuiteStandard) TestMonthIncomeNoTransactions() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	income, err := budget.Income(suite.db, types.NewMonth(2022, 3))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)
}

func (suite *TestSuiteStandard) TestTotalIncomeNoTransactions() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	income, err := budget.TotalIncome(suite.db, types.NewMonth(2031, 3))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)
}

func (suite *TestSuiteStandard) TestTotalBudgetedNoTransactions() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	budgeted, err := budget.TotalBudgeted(suite.db, types.NewMonth(1913, 8))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.IsZero(), "Income is %s, should be 0", budgeted)
}

func (suite *TestSuiteStandard) TestBudgetAvailableDBFail() {
	budget := models.Budget{}

	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	suite.CloseDB()

	_, err = budget.Available(suite.db, types.NewMonth(1990, 1))
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}

func (suite *TestSuiteStandard) TestBudgetIncomeDBFail() {
	budget := models.Budget{}

	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	suite.CloseDB()

	_, err = budget.Income(suite.db, types.NewMonth(1995, 2))
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}

func (suite *TestSuiteStandard) TestBudgetBudgetedDBFail() {
	budget := models.Budget{}

	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	suite.CloseDB()

	_, err = budget.Budgeted(suite.db, types.NewMonth(200, 2))
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}

func (suite *TestSuiteStandard) TestBudgetTotalBudgetedDBFail() {
	budget := models.Budget{}

	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	suite.CloseDB()

	_, err = budget.TotalBudgeted(suite.db, types.NewMonth(2017, 7))
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}

func (suite *TestSuiteStandard) TestBudgetTotalIncomeDBFail() {
	budget := models.Budget{}

	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	suite.CloseDB()

	_, err = budget.TotalIncome(suite.db, types.NewMonth(1300, 12))
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}

func (suite *TestSuiteStandard) TestBudgetOverspentDBFail() {
	budget := models.Budget{}

	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	suite.CloseDB()

	_, err = budget.Overspent(suite.db, types.NewMonth(2023, 9))
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}
