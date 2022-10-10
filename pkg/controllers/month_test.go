package controllers_test

import (
	"fmt"
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
	suite.Assert().Equal(recorder.Header().Get("allow"), "OPTIONS, GET, POST, DELETE")
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestMonth() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID, Name: "Upkeep"})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	account := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID})
	externalAccount := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID, External: true})

	allocationJanuary := suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(20.99),
	})

	allocationFebruary := suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(47.12),
	})

	allocationMarch := suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(31.17),
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
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
									Links: models.EnvelopeMonthLinks{
										Allocation: fmt.Sprintf("http://example.com/v1/allocations/%s", allocationJanuary.Data.ID),
									},
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
									Links: models.EnvelopeMonthLinks{
										Allocation: fmt.Sprintf("http://example.com/v1/allocations/%s", allocationFebruary.Data.ID),
									},
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
									Links: models.EnvelopeMonthLinks{
										Allocation: fmt.Sprintf("http://example.com/v1/allocations/%s", allocationMarch.Data.ID),
									},
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
		suite.assertHTTPStatus(&r, http.StatusOK)
		suite.decodeResponse(&r, &month)

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

		suite.Assert().Equal(expected.Links.Allocation, envelope.Links.Allocation)
	}
}

// TestEnvelopeNoAllocationLink verifies that for an Envelope with no allocation for a specific month,
// the allocation collection endpoint is set as link.
func (suite *TestSuiteStandard) TestEnvelopeNoAllocationLink() {
	var month controllers.MonthResponse

	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	_ = suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", 1), "")
	suite.assertHTTPStatus(&r, http.StatusOK)
	suite.decodeResponse(&r, &month)
	suite.Assert().NotEmpty(month.Data.Categories[0].Envelopes)
	suite.Assert().Equal("http://example.com/v1/allocations", month.Data.Categories[0].Envelopes[0].Links.Allocation)
}

func (suite *TestSuiteStandard) TestEnvelopeAllocationLink() {
	var month controllers.MonthResponse

	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})
	allocation := suite.createTestAllocation(models.AllocationCreate{Amount: decimal.New(1, 1), EnvelopeID: envelope.Data.ID, Month: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", 1), "")
	suite.assertHTTPStatus(&r, http.StatusOK)
	suite.decodeResponse(&r, &month)
	suite.Assert().NotEmpty(month.Data.Categories[0].Envelopes)
	suite.Assert().Equal(allocation.Data.Links.Self, month.Data.Categories[0].Envelopes[0].Links.Allocation)
}

func (suite *TestSuiteStandard) TestMonthNotNil() {
	var month controllers.MonthResponse

	// Verify that the categories list is empty, not nil
	budget := suite.createTestBudget(models.BudgetCreate{})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", 1), "")
	suite.assertHTTPStatus(&r, http.StatusOK)
	suite.decodeResponse(&r, &month)
	if !suite.Assert().NotNil(month.Data.Categories) {
		suite.Assert().FailNow("Categories field is nil, cannot continue")
	}
	suite.Assert().Empty(month.Data.Categories)

	// Verify that the envelopes list is empty, not nil
	_ = suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})

	r = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", 1), "")
	suite.assertHTTPStatus(&r, http.StatusOK)
	suite.decodeResponse(&r, &month)
	suite.Assert().NotNil(month.Data.Categories[0].Envelopes)
	suite.Assert().Empty(month.Data.Categories[0].Envelopes)
}

func (suite *TestSuiteStandard) TestMonthInvalidRequest() {
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/months?month=-56", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/months?budget=noUUID", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	budget := suite.createTestBudget(models.BudgetCreate{})
	r = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "0001-01", 1), "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
	suite.Assert().Equal("The month query parameter must be set", test.DecodeError(suite.T(), r.Body.Bytes()))

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/months?budget=6a463cc8-1938-474a-8aeb-0482b82ffb6f&month=2000-12", "")
	suite.assertHTTPStatus(&r, http.StatusNotFound)
	suite.Assert().Equal("No budget found for the specified ID", test.DecodeError(suite.T(), r.Body.Bytes()))
}

func (suite *TestSuiteStandard) TestMonthDBFail() {
	budget := suite.createTestBudget(models.BudgetCreate{})

	suite.CloseDB()

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", 1), "")
	suite.assertHTTPStatus(&r, http.StatusInternalServerError)
}

func (suite *TestSuiteStandard) TestDeleteMonth() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	allocation1 := suite.createTestAllocation(models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := suite.createTestAllocation(models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope2.Data.ID,
	})

	// Clear allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-01", 1), "")
	suite.assertHTTPStatus(&recorder, http.StatusNoContent)

	// Verify that allocations are deleted
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, allocation1.Data.Links.Self, "")
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)

	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, allocation2.Data.Links.Self, "")
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteMonthFailures() {
	budgetAllocationsLink := suite.createTestBudget(models.BudgetCreate{}).Data.Links.MonthAllocations

	// Bad Request for invalid UUID
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/months?budget=nouuid&month=2022-01", "")
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, budgetAllocationsLink, "")
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/months?budget=059cdead-249f-4f94-8d29-16a80c6b4a09&month=2032-03", "")
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestSetMonthBudgeted() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	allocation1 := suite.createTestAllocation(models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(30),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := suite.createTestAllocation(models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(40),
		EnvelopeID: envelope2.Data.ID,
	})

	// Update in budgeted mode allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-02", 1), controllers.BudgetAllocationMode{Mode: controllers.AllocateLastMonthBudget})
	suite.assertHTTPStatus(&recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	requestString := strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", "2022-02", 1)
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, requestString, "")
	suite.assertHTTPStatus(&recorder, http.StatusOK)
	var envelope1Month controllers.EnvelopeMonthResponse
	suite.decodeResponse(&recorder, &envelope1Month)
	suite.Assert().True(allocation1.Data.Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", allocation1.Data.Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", "2022-02", 1), "")
	suite.assertHTTPStatus(&recorder, http.StatusOK)
	var envelope2Month controllers.EnvelopeMonthResponse
	suite.decodeResponse(&recorder, &envelope2Month)
	suite.Assert().True(allocation2.Data.Amount.Equal(envelope2Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", allocation2.Data.Amount, envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestSetMonthSpend() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	cashAccount := suite.createTestAccount(models.AccountCreate{External: false})
	externalAccount := suite.createTestAccount(models.AccountCreate{External: true})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(30),
		EnvelopeID: envelope1.Data.ID,
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(40),
		EnvelopeID: envelope2.Data.ID,
	})

	eID := &envelope1.Data.ID
	transaction1 := suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 14, 43, 27, 0, time.UTC),
		EnvelopeID:           eID,
		BudgetID:             budget.Data.ID,
		SourceAccountID:      cashAccount.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		Amount:               decimal.NewFromFloat(15),
	})

	// Update in budgeted mode allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-02", 1), controllers.BudgetAllocationMode{Mode: controllers.AllocateLastMonthSpend})
	suite.assertHTTPStatus(&recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	requestString := strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", "2022-02", 1)
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, requestString, "")
	suite.assertHTTPStatus(&recorder, http.StatusOK)
	var envelope1Month controllers.EnvelopeMonthResponse
	suite.decodeResponse(&recorder, &envelope1Month)
	suite.Assert().True(transaction1.Data.Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", transaction1.Data.Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", "2022-02", 1), "")
	suite.assertHTTPStatus(&recorder, http.StatusOK)
	var envelope2Month controllers.EnvelopeMonthResponse
	suite.decodeResponse(&recorder, &envelope2Month)
	suite.Assert().True(envelope2Month.Data.Allocation.Equal(decimal.NewFromFloat(0)), "Expected: 0, got %s, Request ID: %s", envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestSetMonthFailures() {
	budgetAllocationsLink := suite.createTestBudget(models.BudgetCreate{}).Data.Links.MonthAllocations

	// Bad Request for invalid UUID
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/months?budget=nouuid&month=2022-01", "")
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, budgetAllocationsLink, "")
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/months?budget=059cdead-249f-4f94-8d29-16a80c6b4a09&month=2032-03", "")
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)

	// Bad Request for invalid json in body
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budgetAllocationsLink, "YYYY-MM", "2022-01", 1), `{ "mode": INVALID_JSON" }`)
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)

	// Bad Request for invalid mode
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budgetAllocationsLink, "YYYY-MM", "2022-01", 1), `{ "mode": "UNKNOWN_MODE" }`)
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
}
