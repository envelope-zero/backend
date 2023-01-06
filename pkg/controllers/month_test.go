package controllers_test

import (
	"net/http"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v2/internal/types"
	"github.com/envelope-zero/backend/v2/pkg/controllers"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/envelope-zero/backend/v2/test"
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

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.GroupedMonth, "YYYY-MM", "2022-01", -1), "")
	suite.assertHTTPStatus(&r, http.StatusOK)
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
	allocation := suite.createTestAllocation(models.AllocationCreate{Amount: decimal.New(1, 1), EnvelopeID: envelope.Data.ID, Month: types.NewMonth(2022, 1)})

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
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
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
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(30),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
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
	cashAccount := suite.createTestAccount(models.AccountCreate{External: false, OnBudget: true})
	externalAccount := suite.createTestAccount(models.AccountCreate{External: true})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(30),
		EnvelopeID: envelope1.Data.ID,
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
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
