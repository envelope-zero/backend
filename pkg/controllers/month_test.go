package controllers_test

import (
	"net/http"
	"strings"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/test"
	"github.com/shopspring/decimal"
)

func (suite *TestSuiteStandard) TestOptionsMonth() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/months", "")
	suite.Assert().Equal(http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
	suite.Assert().Equal(recorder.Header().Get("allow"), "GET")
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestMonth() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID, Name: "Upkeep"})
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	account := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID})
	externalAccount := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID, External: true})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(20.99),
	})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(47.12),
	})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(31.17),
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 3, 1, 7, 38, 17, 0, time.UTC),
		Amount:               decimal.NewFromFloat(1500),
		Note:                 "Income for march",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      externalAccount.Data.ID,
		DestinationAccountID: account.Data.ID,
		EnvelopeID:           nil,
	})

	tests := []struct {
		path     string
		response controllers.MonthResponse
	}{
		{
			strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", -1),
			controllers.MonthResponse{
				Data: models.Month{
					Month:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
					Income:  decimal.NewFromFloat(0),
					Balance: decimal.NewFromFloat(10.99),
					Categories: []models.CategoryEnvelopes{
						{
							Name: category.Data.Name,
							ID:   category.Data.ID,
							Envelopes: []models.EnvelopeMonth{
								{
									Name:       "Utilities",
									Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
									Spent:      decimal.NewFromFloat(10),
									Balance:    decimal.NewFromFloat(10.99),
									Allocation: decimal.NewFromFloat(20.99),
								},
							},
						},
					},
				},
			},
		},
		{
			strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-02", -1),
			controllers.MonthResponse{
				Data: models.Month{
					Month:   time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
					Income:  decimal.NewFromFloat(0),
					Balance: decimal.NewFromFloat(53.11),
					Categories: []models.CategoryEnvelopes{
						{
							Name: category.Data.Name,
							ID:   category.Data.ID,
							Envelopes: []models.EnvelopeMonth{
								{
									Name:       "Utilities",
									Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
									Balance:    decimal.NewFromFloat(53.11),
									Spent:      decimal.NewFromFloat(5),
									Allocation: decimal.NewFromFloat(47.12),
								},
							},
						},
					},
				},
			},
		},
		{
			strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-03", -1),
			controllers.MonthResponse{
				Data: models.Month{
					Month:   time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
					Income:  decimal.NewFromFloat(1500),
					Balance: decimal.NewFromFloat(69.28),
					Categories: []models.CategoryEnvelopes{
						{
							Name: category.Data.Name,
							ID:   category.Data.ID,
							Envelopes: []models.EnvelopeMonth{
								{
									Name:       "Utilities",
									Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
									Balance:    decimal.NewFromFloat(69.28),
									Spent:      decimal.NewFromFloat(15),
									Allocation: decimal.NewFromFloat(31.17),
								},
							},
						},
					},
				},
			},
		},
	}

	var month controllers.MonthResponse
	for _, tt := range tests {
		r := test.Request(suite.controller, suite.T(), http.MethodGet, tt.path, "")
		test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
		test.DecodeResponse(suite.T(), &r, &month)

		// Verify income calculation
		suite.Assert().True(month.Data.Income.Equal(tt.response.Data.Income))

		// Verify month balance calculation
		suite.Assert().True(month.Data.Balance.Equal(tt.response.Data.Balance), "Month balance calculation for %v is wrong: should be %v, but is %v: %#v", month.Data.Month, tt.response.Data.Balance, month.Data.Balance, month.Data)

		if !suite.Assert().Len(month.Data.Categories, 1) {
			suite.Assert().FailNow("Response category length does not match!", "Category list does not have exactly 1 item, it has %d, Request ID: %s", len(month.Data.Categories), r.Header().Get("x-request-id"))
		}

		if !suite.Assert().Len(month.Data.Categories[0].Envelopes, 1) {
			suite.Assert().FailNow("Response envelope length does not match!", "Envelope list does not have exactly 1 item, it has %d, Request ID: %s", len(month.Data.Categories[0].Envelopes), r.Header().Get("x-request-id"))
		}

		expected := tt.response.Data.Categories[0].Envelopes[0]
		envelope := month.Data.Categories[0].Envelopes[0]
		suite.Assert().True(envelope.Spent.Equal(expected.Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", month.Data.Month, expected.Spent, envelope.Spent, month.Data)
		suite.Assert().True(envelope.Balance.Equal(expected.Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", month.Data.Month, expected.Balance, envelope.Balance, month.Data)
		suite.Assert().True(envelope.Allocation.Equal(expected.Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", month.Data.Month, expected.Allocation, envelope.Allocation, month.Data)
	}
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestMonthNotNil() {
	var month controllers.MonthResponse

	// Verify that the categories list is empty, not nil
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
	test.DecodeResponse(suite.T(), &r, &month)
	if !suite.Assert().NotNil(month.Data.Categories) {
		suite.Assert().FailNow("Categories field is nil, cannot continue")
	}
	suite.Assert().Empty(month.Data.Categories)

	// Verify that the envelopes list is empty, not nil
	_ = suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})

	r = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
	test.DecodeResponse(suite.T(), &r, &month)
	suite.Assert().NotNil(month.Data.Categories[0].Envelopes)
	suite.Assert().Empty(month.Data.Categories[0].Envelopes)
}

func (suite *TestSuiteStandard) TestMonthInvalidRequest() {
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/months?month=-56", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/months?budget=noUUID", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	r = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "0001-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
	suite.Assert().Equal("You cannot request data for no month", test.DecodeError(suite.T(), r.Body.Bytes()))

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/months?budget=6a463cc8-1938-474a-8aeb-0482b82ffb6f", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusNotFound)
	suite.Assert().Equal("No budget found for the specified ID", test.DecodeError(suite.T(), r.Body.Bytes()))
}

func (suite *TestSuiteStandard) TestMonthDBFail() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})

	suite.CloseDB()

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusInternalServerError)
}
