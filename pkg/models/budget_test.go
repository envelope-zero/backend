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

	budget, err = budget.WithCalculations(suite.db)
	assert.Nil(suite.T(), err)

	shouldBalance := decimal.NewFromFloat(7269.38)
	assert.True(suite.T(), budget.Balance.Equal(shouldBalance), "Balance for budget is not correct. Should be %s, is %s", shouldBalance, budget.Balance)

	// Verify income for used budget in March
	shouldIncome := decimal.NewFromFloat(4600)
	income, err := budget.Income(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.Equal(shouldIncome), "Income is %s, should be %s", income, shouldIncome)

	// Verify income for empty budget in March
	income, err = emptyBudget.Income(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), income.IsZero(), "Income is %s, should be 0", income)

	// Verify budgeted for used budget
	budgeted, err := budget.Allocated(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.Equal(decimal.NewFromFloat(25)), "Budgeted is %s, should be 25", budgeted)

	// Verify budgeted for empty budget
	budgeted, err = emptyBudget.Allocated(suite.db, marchTwentyTwentyTwo)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), budgeted.IsZero(), "Budgeted is %s, should be 0", budgeted)
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

	_, err = budget.Allocated(suite.db, types.NewMonth(200, 2))
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestMonth() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.ID, Name: "Upkeep"})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.ID, Name: "Utilities"})
	account := suite.createTestAccount(models.AccountCreate{BudgetID: budget.ID, OnBudget: true, Name: "TestMonth"})
	externalAccount := suite.createTestAccount(models.AccountCreate{BudgetID: budget.ID, External: true})

	allocationJanuary := suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.ID,
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(20.99),
	})

	allocationFebruary := suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.ID,
		Month:      types.NewMonth(2022, 2),
		Amount:     decimal.NewFromFloat(47.12),
	})

	allocationMarch := suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.ID,
		Month:      types.NewMonth(2022, 3),
		Amount:     decimal.NewFromFloat(31.17),
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.ID,
		SourceAccountID:      account.ID,
		DestinationAccountID: externalAccount.ID,
		EnvelopeID:           &envelope.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.ID,
		SourceAccountID:      account.ID,
		DestinationAccountID: externalAccount.ID,
		EnvelopeID:           &envelope.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.ID,
		SourceAccountID:      account.ID,
		DestinationAccountID: externalAccount.ID,
		EnvelopeID:           &envelope.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 3, 1, 7, 38, 17, 0, time.UTC),
		Amount:               decimal.NewFromFloat(1500),
		Note:                 "Income for march",
		BudgetID:             budget.ID,
		SourceAccountID:      externalAccount.ID,
		DestinationAccountID: account.ID,
		EnvelopeID:           nil,
	})

	tests := []struct {
		month  types.Month
		result models.Month
	}{
		{
			types.NewMonth(2022, 1),
			models.Month{
				Month:      types.NewMonth(2022, 1),
				Income:     decimal.NewFromFloat(0),
				Balance:    decimal.NewFromFloat(10.99),
				Spent:      decimal.NewFromFloat(-10),
				Allocation: decimal.NewFromFloat(20.99),
				Available:  decimal.NewFromFloat(-20.99),
				Categories: []models.CategoryEnvelopes{
					{
						Category:   category,
						Balance:    decimal.NewFromFloat(10.99),
						Spent:      decimal.NewFromFloat(-10),
						Allocation: decimal.NewFromFloat(20.99),
						Envelopes: []models.EnvelopeMonth{
							{
								Envelope:   envelope,
								Month:      types.NewMonth(2022, 1),
								Spent:      decimal.NewFromFloat(-10),
								Balance:    decimal.NewFromFloat(10.99),
								Allocation: decimal.NewFromFloat(20.99),
								Links: models.EnvelopeMonthLinks{
									Allocation: fmt.Sprintf("http://example.com/v1/allocations/%s", allocationJanuary.ID),
								},
							},
						},
					},
				},
			},
		},
		{
			types.NewMonth(2022, 2),
			models.Month{
				Month:      types.NewMonth(2022, 2),
				Income:     decimal.NewFromFloat(0),
				Balance:    decimal.NewFromFloat(53.11),
				Spent:      decimal.NewFromFloat(-5),
				Allocation: decimal.NewFromFloat(47.12),
				Available:  decimal.NewFromFloat(-68.11),
				Categories: []models.CategoryEnvelopes{
					{
						Category:   category,
						Balance:    decimal.NewFromFloat(53.11),
						Spent:      decimal.NewFromFloat(-5),
						Allocation: decimal.NewFromFloat(47.12),
						Envelopes: []models.EnvelopeMonth{
							{
								Envelope:   envelope,
								Month:      types.NewMonth(2022, 2),
								Balance:    decimal.NewFromFloat(53.11),
								Spent:      decimal.NewFromFloat(-5),
								Allocation: decimal.NewFromFloat(47.12),
								Links: models.EnvelopeMonthLinks{
									Allocation: fmt.Sprintf("http://example.com/v1/allocations/%s", allocationFebruary.ID),
								},
							},
						},
					},
				},
			},
		},
		{
			types.NewMonth(2022, 3),
			models.Month{
				Month:      types.NewMonth(2022, 3),
				Income:     decimal.NewFromFloat(1500),
				Balance:    decimal.NewFromFloat(69.28),
				Spent:      decimal.NewFromFloat(-15),
				Allocation: decimal.NewFromFloat(31.17),
				Available:  decimal.NewFromFloat(1400.72),
				Categories: []models.CategoryEnvelopes{
					{
						Category:   category,
						Balance:    decimal.NewFromFloat(69.28),
						Spent:      decimal.NewFromFloat(-15),
						Allocation: decimal.NewFromFloat(31.17),
						Envelopes: []models.EnvelopeMonth{
							{
								Envelope:   envelope,
								Month:      types.NewMonth(2022, 3),
								Balance:    decimal.NewFromFloat(69.28),
								Spent:      decimal.NewFromFloat(-15),
								Allocation: decimal.NewFromFloat(31.17),
								Links: models.EnvelopeMonthLinks{
									Allocation: fmt.Sprintf("http://example.com/v1/allocations/%s", allocationMarch.ID),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.month.String(), func(t *testing.T) {
			month, err := budget.Month(suite.db, tt.month, "http://example.com")
			assert.Nil(t, err)

			// Verify income calculation
			assert.True(t, month.Income.Equal(tt.result.Income))

			// Verify month balance calculation
			assert.True(t, month.Balance.Equal(tt.result.Balance), "Month balance calculation for %v is wrong: should be %v, but is %v: %#v", month.Month, tt.result.Balance, month.Balance, month)

			// Verify allocation calculation
			assert.True(t, month.Allocation.Equal(tt.result.Allocation), "Month allocation sum for %v is wrong: should be %v, but is %v: %#v", month.Month, tt.result.Allocation, month.Allocation, month)

			// Verify available calculation
			assert.True(t, month.Available.Equal(tt.result.Available), "Month available sum for %v is wrong: should be %v, but is %v: %#v", month.Month, tt.result.Available, month.Available, month)

			// Verify month spent calculation
			assert.True(t, month.Spent.Equal(tt.result.Spent), "Month spent is wrong. Should be %v, but is %v: %#v", tt.result.Spent, month.Spent, month)

			if !suite.Assert().Len(month.Categories, 1) {
				suite.Assert().FailNow("Response category length does not match!", "Category list does not have exactly 1 item, it has %d, Request ID: %s", len(month.Categories))
			}

			if !suite.Assert().Len(month.Categories[0].Envelopes, 1) {
				suite.Assert().FailNow("Response envelope length does not match!", "Envelope list does not have exactly 1 item, it has %d, Request ID: %s", len(month.Categories[0].Envelopes))
			}

			// Category calculations
			expectedCategory := tt.result.Categories[0]
			category := month.Categories[0]

			assert.True(t, category.Spent.Equal(expectedCategory.Spent), "Monthly category spent calculation for %v is wrong: should be %v, but is %v: %#v", month.Month, expectedCategory.Spent, category.Spent, month)
			assert.True(t, category.Balance.Equal(expectedCategory.Balance), "Monthly category balance calculation for %v is wrong: should be %v, but is %v: %#v", month.Month, expectedCategory.Balance, category.Balance, month)
			assert.True(t, category.Allocation.Equal(expectedCategory.Allocation), "Monthly category allocation fetch for %v is wrong: should be %v, but is %v: %#v", month.Month, expectedCategory.Allocation, category.Allocation, month)

			// Envelope calculation
			expectedEnvelope := tt.result.Categories[0].Envelopes[0]
			envelope := month.Categories[0].Envelopes[0]

			assert.True(t, envelope.Spent.Equal(expectedEnvelope.Spent), "Monthly envelope spent calculation for %v is wrong: should be %v, but is %v: %#v", month.Month, expectedEnvelope.Spent, envelope.Spent, month)
			assert.True(t, envelope.Balance.Equal(expectedEnvelope.Balance), "Monthly envelope balance calculation for %v is wrong: should be %v, but is %v: %#v", month.Month, expectedEnvelope.Balance, envelope.Balance, month)
			assert.True(t, envelope.Allocation.Equal(expectedEnvelope.Allocation), "Monthly envelope allocation fetch for %v is wrong: should be %v, but is %v: %#v", month.Month, expectedEnvelope.Allocation, envelope.Allocation, month)

			suite.Assert().Equal(expectedEnvelope.Links.Allocation, envelope.Links.Allocation)
		})
	}
}
