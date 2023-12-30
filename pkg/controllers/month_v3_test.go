package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestMonthsV3Get() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", -1), "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)
}

func (suite *TestSuiteStandard) TestMonthsGetV3EnvelopeAllocationLink() {
	var month controllers.MonthResponseV3

	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID})
	_ = suite.createTestAllocation(models.AllocationCreate{Amount: decimal.NewFromFloat(10), EnvelopeID: envelope.Data.ID, Month: types.NewMonth(2022, 1)})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)
	suite.decodeResponse(&r, &month)
	suite.Assert().NotEmpty(month.Data.Categories[0].Envelopes)
	suite.Assert().True(month.Data.Categories[0].Allocation.Equal(decimal.NewFromFloat(10)))
}

func (suite *TestSuiteStandard) TestMonthsGetV3NotNil() {
	var month controllers.MonthResponseV3

	// Verify that the categories list is empty, not nil
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)
	suite.decodeResponse(&r, &month)
	if !suite.Assert().NotNil(month.Data.Categories) {
		suite.Assert().FailNow("Categories field is nil, cannot continue")
	}
	suite.Assert().Empty(month.Data.Categories)

	// Verify that the envelopes list is empty, not nil
	_ = suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: budget.Data.ID})

	r = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)
	suite.decodeResponse(&r, &month)
	suite.Assert().NotNil(month.Data.Categories[0].Envelopes)
	suite.Assert().Empty(month.Data.Categories[0].Envelopes)
}

func (suite *TestSuiteStandard) TestMonthsGetV3InvalidRequest() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	tests := []struct {
		name     string                                          // name of the test
		path     string                                          // path to request
		testFunc func(t *testing.T, r httptest.ResponseRecorder) // additional tests
		status   int                                             // expected status
	}{
		{"Invalid month", "http://example.com/v3/months?month=-56", nil, http.StatusBadRequest},
		{"Invalid UUID", "http://example.com/v3/months?budget=noUUID", nil, http.StatusBadRequest},
		{"Month query parameter not set", strings.Replace(budget.Data.Links.Month, "YYYY-MM", "0001-01", 1), func(t *testing.T, r httptest.ResponseRecorder) {
			assert.Equal(t, "the month query parameter must be set", test.DecodeError(suite.T(), r.Body.Bytes()))
		}, http.StatusBadRequest},
		{"No budget with ID", "http://example.com/v3/months?budget=6a463cc8-1938-474a-8aeb-0482b82ffb6f&month=2000-12", func(t *testing.T, r httptest.ResponseRecorder) {
			assert.Equal(t, "there is no Budget with this ID", test.DecodeError(suite.T(), r.Body.Bytes()))
		}, http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodGet, tt.path, "")
			assertHTTPStatus(t, &r, tt.status)

			if tt.testFunc != nil {
				tt.testFunc(t, r)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestMonthsGetV3DBFail() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	suite.CloseDB()

	r := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	assertHTTPStatus(suite.T(), &r, http.StatusInternalServerError)
}

func (suite *TestSuiteStandard) TestMonthsGetV3Delete() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID})

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
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify that allocations are deleted
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, allocation1.Data.Links.Self, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)

	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, allocation2.Data.Links.Self, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestMonthsV3DeleteFail() {
	b := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	// Bad Request for invalid UUID
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v3/months?budget=nouuid&month=2022-01", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, fmt.Sprintf("http://example.com/v3/months?budget=%s&month=022-01", b.Data.ID), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v3/months?budget=059cdead-249f-4f94-8d29-16a80c6b4a09&month=2032-03", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestMonthsV3AllocateBudgeted() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID})
	archivedEnvelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID, Archived: true})

	e1Amount := decimal.NewFromFloat(30)
	e2Amount := decimal.NewFromFloat(40)
	eArchivedAmount := decimal.NewFromFloat(50)

	january := types.NewMonth(2022, 1)
	february := january.AddDate(0, 1)

	// Allocate funds to the months
	// TODO: Replace this with createTestMonthConfigV3 once Allocations are integrated and not transparently created by API v3
	allocations := []struct {
		envelopeMonth string
		month         types.Month
		amount        decimal.Decimal
	}{
		{envelope1.Data.Links.Month, january, e1Amount},
		{envelope2.Data.Links.Month, january, e2Amount},
		{archivedEnvelope.Data.Links.Month, january, eArchivedAmount},
	}

	for _, allocation := range allocations {
		recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, strings.Replace(allocation.envelopeMonth, "YYYY-MM", january.String(), 1), map[string]string{
			"allocation": allocation.amount.String(),
		})
		assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
		var a controllers.MonthConfigResponseV3
		suite.decodeResponse(&recorder, &a)
		assert.True(suite.T(), allocation.amount.Equal(a.Data.Allocation))
	}

	// Update in budgeted mode allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.Month, "YYYY-MM", february.String(), 1), controllers.BudgetAllocationMode{Mode: controllers.AllocateLastMonthBudget})
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", february.String(), 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope1Month controllers.MonthConfigResponseV3
	suite.decodeResponse(&recorder, &envelope1Month)
	suite.Assert().True(e1Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", e1Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", february.String(), 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope2Month controllers.MonthConfigResponseV3
	suite.decodeResponse(&recorder, &envelope2Month)
	suite.Assert().True(e2Amount.Equal(envelope2Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", e2Amount, envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the archived envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(archivedEnvelope.Data.Links.Month, "YYYY-MM", february.String(), 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var archivedEnvelopeMonth controllers.MonthConfigResponseV3
	suite.decodeResponse(&recorder, &archivedEnvelopeMonth)

	// Quick allocations skip archived envelopes, so this should be zero
	suite.Assert().True(archivedEnvelopeMonth.Data.Allocation.IsZero(), "Expected: 0, got %s, Request ID: %s", archivedEnvelopeMonth.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestMonthsV3AllocateSpend() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	cashAccount := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{External: false, OnBudget: true, Name: "TestSetMonthSpend Cash"})
	externalAccount := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{External: true, Name: "TestSetMonthSpend External"})
	category := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID})

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
	transaction1 := suite.createTestTransactionV3(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 14, 43, 27, 0, time.UTC),
		EnvelopeID:           eID,
		BudgetID:             budget.Data.ID,
		SourceAccountID:      cashAccount.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		Amount:               decimal.NewFromFloat(15),
	})

	// Update in budgeted mode allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-02", 1), controllers.BudgetAllocationMode{Mode: controllers.AllocateLastMonthSpend})
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	requestString := strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", "2022-02", 1)
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, requestString, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope1Month controllers.EnvelopeMonthResponseV3
	suite.decodeResponse(&recorder, &envelope1Month)

	// We allocated by the spend of the month before, so the allocation should equal the amount of the transaction
	suite.Assert().True(transaction1.Data.Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", transaction1.Data.Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", "2022-02", 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope2Month controllers.EnvelopeMonthResponseV3
	suite.decodeResponse(&recorder, &envelope2Month)

	// No spend on this envelope in January, therefore no allocation in february
	suite.Assert().True(envelope2Month.Data.Allocation.Equal(decimal.NewFromFloat(0)), "Expected: 0, got %s, Request ID: %s", envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestMonthsV3PostFails() {
	budgetAllocationsLink := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{}).Data.Links.Month

	tests := []struct {
		name   string
		url    string
		body   string
		status int // expected HTTP status
	}{
		{"Invalid UUID", "http://example.com/v3/months?budget=nouuid&month=2022-01", "", http.StatusBadRequest},
		{"Invalid month", budgetAllocationsLink, "", http.StatusBadRequest},
		{"Non-existing budget", "http://example.com/v3/months?budget=059cdead-249f-4f94-8d29-16a80c6b4a09&month=2032-03", "", http.StatusNotFound},
		{"Invalid body", strings.Replace(budgetAllocationsLink, "YYYY-MM", "2022-01", 1), `{ "mode": INVALID_JSON" }`, http.StatusBadRequest},
		{"Invalid mode", strings.Replace(budgetAllocationsLink, "YYYY-MM", "2022-01", 1), `{ "mode": "UNKNOWN_MODE" }`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			recorder := test.Request(suite.controller, t, http.MethodPost, tt.url, tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestMonthsV3 verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestMonthsV3() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: budget.Data.ID, Name: "Upkeep"})
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: category.Data.ID, Name: "Utilities"})
	account := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: budget.Data.ID, OnBudget: true, Name: "TestMonth"})
	externalAccount := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: budget.Data.ID, External: true})

	// Allocate funds to the months
	// TODO: Replace this with createTestMonthConfigV3 once Allocations are integrated and not transparently created by API v3
	allocations := []struct {
		month  types.Month
		amount decimal.Decimal
	}{
		{types.NewMonth(2022, 1), decimal.NewFromFloat(20.99)},
		{types.NewMonth(2022, 2), decimal.NewFromFloat(47.12)},
		{types.NewMonth(2022, 3), decimal.NewFromFloat(31.17)},
	}

	for _, allocation := range allocations {
		recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, strings.Replace(envelope.Data.Links.Month, "YYYY-MM", allocation.month.String(), 1), map[string]string{
			"allocation": allocation.amount.String(),
		})
		assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
		var a controllers.MonthConfigResponseV3
		suite.decodeResponse(&recorder, &a)
		assert.True(suite.T(), allocation.amount.Equal(a.Data.Allocation))
	}

	_ = suite.createTestTransactionV3(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransactionV3(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransactionV3(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransactionV3(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 3, 1, 7, 38, 17, 0, time.UTC),
		Amount:               decimal.NewFromFloat(1500),
		Note:                 "Income for march",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      externalAccount.Data.ID,
		DestinationAccountID: account.Data.ID,
		EnvelopeID:           nil,
	})

	tests := []struct {
		month  types.Month
		result controllers.MonthV3
	}{
		{
			types.NewMonth(2022, 1),
			controllers.MonthV3{
				Month:      types.NewMonth(2022, 1),
				Income:     decimal.NewFromFloat(0),
				Balance:    decimal.NewFromFloat(10.99),
				Spent:      decimal.NewFromFloat(-10),
				Allocation: decimal.NewFromFloat(20.99),
				Available:  decimal.NewFromFloat(-20.99),
				Categories: []controllers.CategoryEnvelopesV3{
					{
						Category:   category.Data.Category,
						Balance:    decimal.NewFromFloat(10.99),
						Spent:      decimal.NewFromFloat(-10),
						Allocation: decimal.NewFromFloat(20.99),
						Envelopes: []controllers.EnvelopeMonthV3{
							{
								Envelope:   envelope.Data.Envelope,
								Spent:      decimal.NewFromFloat(-10),
								Balance:    decimal.NewFromFloat(10.99),
								Allocation: decimal.NewFromFloat(20.99),
							},
						},
					},
				},
			},
		},
		{
			types.NewMonth(2022, 2),
			controllers.MonthV3{
				Month:      types.NewMonth(2022, 2),
				Income:     decimal.NewFromFloat(0),
				Balance:    decimal.NewFromFloat(53.11),
				Spent:      decimal.NewFromFloat(-5),
				Allocation: decimal.NewFromFloat(47.12),
				Available:  decimal.NewFromFloat(-68.11),
				Categories: []controllers.CategoryEnvelopesV3{
					{
						Category:   category.Data.Category,
						Balance:    decimal.NewFromFloat(53.11),
						Spent:      decimal.NewFromFloat(-5),
						Allocation: decimal.NewFromFloat(47.12),
						Envelopes: []controllers.EnvelopeMonthV3{
							{
								Envelope:   envelope.Data.Envelope,
								Balance:    decimal.NewFromFloat(53.11),
								Spent:      decimal.NewFromFloat(-5),
								Allocation: decimal.NewFromFloat(47.12),
							},
						},
					},
				},
			},
		},
		{
			types.NewMonth(2022, 3),
			controllers.MonthV3{
				Month:      types.NewMonth(2022, 3),
				Income:     decimal.NewFromFloat(1500),
				Balance:    decimal.NewFromFloat(69.28),
				Spent:      decimal.NewFromFloat(-15),
				Allocation: decimal.NewFromFloat(31.17),
				Available:  decimal.NewFromFloat(1400.72),
				Categories: []controllers.CategoryEnvelopesV3{
					{
						Category:   category.Data.Category,
						Balance:    decimal.NewFromFloat(69.28),
						Spent:      decimal.NewFromFloat(-15),
						Allocation: decimal.NewFromFloat(31.17),
						Envelopes: []controllers.EnvelopeMonthV3{
							{
								Envelope:   envelope.Data.Envelope,
								Balance:    decimal.NewFromFloat(69.28),
								Spent:      decimal.NewFromFloat(-15),
								Allocation: decimal.NewFromFloat(31.17),
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.month.String(), func(t *testing.T) {
			// Get month data
			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", tt.month.String(), 1), "")
			assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

			// Parse month data
			var response controllers.MonthResponseV3
			suite.decodeResponse(&recorder, &response)
			month := response.Data

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

			// Verify the links are set correctly
			assert.Equal(t, envelope.Data.Links.Month, month.Categories[0].Envelopes[0].Links.Month)

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
		})
	}
}
