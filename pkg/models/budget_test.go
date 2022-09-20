package models_test

import (
	"time"

	"github.com/envelope-zero/backend/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestBudgetCalculations() {
	marchFifteenthTwentyTwentyTwo := time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC)

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
			Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
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
			Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
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
			Month:      marchFifteenthTwentyTwentyTwo,
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
			Month:      time.Date(2170, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	err = suite.db.Save(&allocationFuture).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	salaryTransactionFebruary := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 marchFifteenthTwentyTwentyTwo,
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
			Date:                 marchFifteenthTwentyTwentyTwo,
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
			Date:                 marchFifteenthTwentyTwentyTwo.AddDate(0, 1, 0),
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
			Date:                 marchFifteenthTwentyTwentyTwo,
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
			Date:                 marchFifteenthTwentyTwentyTwo,
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
			Date:                 marchFifteenthTwentyTwentyTwo,
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
	income, err := budget.Income(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.Equal(shouldIncome), "Income is %s, should be %s", income, shouldIncome)

	// Verify total income for used budget
	shouldIncomeTotal := decimal.NewFromFloat(4600)
	income, err = budget.TotalIncome(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.Equal(shouldIncomeTotal), "Income is %s, should be %s", income, shouldIncomeTotal)

	// Verify income for empty budget in March
	income, err = emptyBudget.Income(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)

	// Verify total income for empty budget
	income, err = emptyBudget.TotalIncome(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)

	// Verify total budgeted for used budget
	budgeted, err := budget.TotalBudgeted(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.Equal(decimal.NewFromFloat(67)), "Budgeted is %s, should be 67", budgeted)

	// Verify budgeted for used budget
	budgeted, err = budget.Budgeted(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.Equal(decimal.NewFromFloat(25)), "Budgeted is %s, should be 25", budgeted)

	// Verify total budgeted for empty budget
	budgeted, err = emptyBudget.TotalBudgeted(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.IsZero(), "Budgeted is %s, should be 0", budgeted)

	// Verify budgeted for empty budget
	budgeted, err = emptyBudget.Budgeted(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.IsZero(), "Budgeted is %s, should be 0", budgeted)

	// Verify overspent calculation for month without spend
	overspent, err := budget.Overspent(suite.db, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), overspent.IsZero(), "Overspent is %s, should be 0", overspent)

	// Verify overspent calculation for month with spend
	overspent, err = budget.Overspent(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), overspent.Equal(decimal.NewFromFloat(20)), "Overspent is %s, should be 20", overspent)

	available, err := budget.Available(suite.db, marchFifteenthTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), available.Equal(decimal.NewFromFloat(4513)), "Available is %s, should be 4513", available)
}

func (suite *TestSuiteStandard) TestMonthIncomeNoTransactions() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	income, err := budget.Income(suite.db, time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)
}

func (suite *TestSuiteStandard) TestTotalIncomeNoTransactions() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	income, err := budget.TotalIncome(suite.db, time.Date(2031, 3, 17, 0, 0, 0, 0, time.UTC))
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)
}

func (suite *TestSuiteStandard) TestTotalBudgetedNoTransactions() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	budgeted, err := budget.TotalBudgeted(suite.db, time.Date(1913, 8, 3, 0, 0, 0, 0, time.UTC))
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

	_, err = budget.Available(suite.db, time.Now())
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

	_, err = budget.Income(suite.db, time.Now())
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

	_, err = budget.Budgeted(suite.db, time.Now())
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

	_, err = budget.TotalBudgeted(suite.db, time.Now())
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

	_, err = budget.TotalIncome(suite.db, time.Now())
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

	_, err = budget.Overspent(suite.db, time.Now())
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}
