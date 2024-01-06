package v3_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	v3 "github.com/envelope-zero/backend/v4/pkg/controllers/v3"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestMonthsGet() {
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})

	r := test.Request(suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", -1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
}

func (suite *TestSuiteStandard) TestMonthsGetEnvelopeAllocationLink() {
	var month v3.MonthResponse

	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})
	category := suite.createTestCategory(suite.T(), v3.CategoryEditable{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID})

	_ = suite.patchTestMonthConfig(suite.T(),
		envelope.Data.ID,
		types.NewMonth(2022, 1),
		models.MonthConfigCreate{
			Allocation: decimal.NewFromFloat(10),
		})

	r := test.Request(suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
	test.DecodeResponse(suite.T(), &r, &month)
	suite.Assert().NotEmpty(month.Data.Categories[0].Envelopes)
	suite.Assert().True(month.Data.Categories[0].Allocation.Equal(decimal.NewFromFloat(10)))
}

func (suite *TestSuiteStandard) TestMonthsGetNotNil() {
	var month v3.MonthResponse

	// Verify that the categories list is empty, not nil
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})

	r := test.Request(suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
	test.DecodeResponse(suite.T(), &r, &month)
	if !suite.Assert().NotNil(month.Data.Categories) {
		suite.Assert().FailNow("Categories field is nil, cannot continue")
	}
	suite.Assert().Empty(month.Data.Categories)

	// Verify that the envelopes list is empty, not nil
	_ = suite.createTestCategory(suite.T(), v3.CategoryEditable{BudgetID: budget.Data.ID})

	r = test.Request(suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
	test.DecodeResponse(suite.T(), &r, &month)
	suite.Assert().NotNil(month.Data.Categories[0].Envelopes)
	suite.Assert().Empty(month.Data.Categories[0].Envelopes)
}

func (suite *TestSuiteStandard) TestMonthsGetInvalidRequest() {
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})

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
			r := test.Request(t, http.MethodGet, tt.path, "")
			test.AssertHTTPStatus(t, &r, tt.status)

			if tt.testFunc != nil {
				tt.testFunc(t, r)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestMonthsGetDBFail() {
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})

	suite.CloseDB()

	r := test.Request(suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusInternalServerError)
}

func (suite *TestSuiteStandard) TestMonthsGetDelete() {
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})
	category := suite.createTestCategory(suite.T(), v3.CategoryEditable{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID})

	monthConfig1 := suite.patchTestMonthConfig(suite.T(),
		envelope1.Data.ID,
		types.NewMonth(2022, 1),
		models.MonthConfigCreate{Allocation: decimal.NewFromFloat(15.42)},
	)

	monthConfig2 := suite.patchTestMonthConfig(suite.T(),
		envelope2.Data.ID,
		types.NewMonth(2022, 1),
		models.MonthConfigCreate{Allocation: decimal.NewFromFloat(15.42)},
	)

	// Clear allocations
	recorder := test.Request(suite.T(), http.MethodDelete, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify that allocations are deleted
	recorder = test.Request(suite.T(), http.MethodGet, monthConfig1.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var response v3.MonthConfigResponse
	test.DecodeResponse(suite.T(), &recorder, &response)
	assert.True(suite.T(), response.Data.Allocation.IsZero(), "Allocation is not zero after deletion")

	recorder = test.Request(suite.T(), http.MethodGet, monthConfig2.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	test.DecodeResponse(suite.T(), &recorder, &response)
	assert.True(suite.T(), response.Data.Allocation.IsZero(), "Allocation is not zero after deletion")
}

func (suite *TestSuiteStandard) TestMonthsDeleteFail() {
	b := suite.createTestBudget(suite.T(), v3.BudgetEditable{})

	// Bad Request for invalid UUID
	recorder := test.Request(suite.T(), http.MethodDelete, "http://example.com/v3/months?budget=nouuid&month=2022-01", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.T(), http.MethodDelete, fmt.Sprintf("http://example.com/v3/months?budget=%s&month=022-01", b.Data.ID), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.T(), http.MethodDelete, "http://example.com/v3/months?budget=059cdead-249f-4f94-8d29-16a80c6b4a09&month=2032-03", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestMonthsAllocateBudgeted() {
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})
	category := suite.createTestCategory(suite.T(), v3.CategoryEditable{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID})
	archivedEnvelope := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID, Archived: true})

	e1Amount := decimal.NewFromFloat(30)
	e2Amount := decimal.NewFromFloat(40)
	eArchivedAmount := decimal.NewFromFloat(50)

	january := types.NewMonth(2022, 1)
	february := january.AddDate(0, 1)

	// Allocate funds to the months
	allocations := []struct {
		envelopeID uuid.UUID
		month      types.Month
		amount     decimal.Decimal
	}{
		{envelope1.Data.ID, january, e1Amount},
		{envelope2.Data.ID, january, e2Amount},
		{archivedEnvelope.Data.ID, january, eArchivedAmount},
	}

	for _, allocation := range allocations {
		suite.patchTestMonthConfig(suite.T(), allocation.envelopeID, allocation.month, models.MonthConfigCreate{
			Allocation: allocation.amount,
		})
	}

	// Update in budgeted mode allocations
	recorder := test.Request(suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.Month, "YYYY-MM", february.String(), 1), v3.BudgetAllocationMode{Mode: v3.AllocateLastMonthBudget})
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	recorder = test.Request(suite.T(), http.MethodGet, strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", february.String(), 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope1Month v3.MonthConfigResponse
	test.DecodeResponse(suite.T(), &recorder, &envelope1Month)
	suite.Assert().True(e1Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", e1Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", february.String(), 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope2Month v3.MonthConfigResponse
	test.DecodeResponse(suite.T(), &recorder, &envelope2Month)
	suite.Assert().True(e2Amount.Equal(envelope2Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", e2Amount, envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the archived envelope
	recorder = test.Request(suite.T(), http.MethodGet, strings.Replace(archivedEnvelope.Data.Links.Month, "YYYY-MM", february.String(), 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var archivedEnvelopeMonth v3.MonthConfigResponse
	test.DecodeResponse(suite.T(), &recorder, &archivedEnvelopeMonth)

	// Quick allocations skip archived envelopes, so this should be zero
	suite.Assert().True(archivedEnvelopeMonth.Data.Allocation.IsZero(), "Expected: 0, got %s, Request ID: %s", archivedEnvelopeMonth.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestMonthsAllocateSpend() {
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})
	cashAccount := suite.createTestAccount(suite.T(), models.Account{External: false, OnBudget: true, Name: "TestSetMonthSpend Cash"})
	externalAccount := suite.createTestAccount(suite.T(), models.Account{External: true, Name: "TestSetMonthSpend External"})
	category := suite.createTestCategory(suite.T(), v3.CategoryEditable{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID})

	_ = suite.patchTestMonthConfig(suite.T(),
		envelope1.Data.ID,
		types.NewMonth(2022, 1),
		models.MonthConfigCreate{Allocation: decimal.NewFromFloat(30)},
	)

	_ = suite.patchTestMonthConfig(suite.T(),
		envelope2.Data.ID,
		types.NewMonth(2022, 1),
		models.MonthConfigCreate{Allocation: decimal.NewFromFloat(40)},
	)

	eID := &envelope1.Data.ID
	transaction1 := suite.createTestTransaction(suite.T(), models.Transaction{
		Date:                 time.Date(2022, 1, 15, 14, 43, 27, 0, time.UTC),
		EnvelopeID:           eID,
		BudgetID:             budget.Data.ID,
		SourceAccountID:      cashAccount.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		Amount:               decimal.NewFromFloat(15),
	})

	// Update in budgeted mode allocations
	recorder := test.Request(suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.Month, "YYYY-MM", "2022-02", 1), v3.BudgetAllocationMode{Mode: v3.AllocateLastMonthSpend})
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	requestString := strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", "2022-02", 1)
	recorder = test.Request(suite.T(), http.MethodGet, requestString, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope1Month v3.MonthConfigResponse
	test.DecodeResponse(suite.T(), &recorder, &envelope1Month)

	// We allocated by the spend of the month before, so the allocation should equal the amount of the transaction
	suite.Assert().True(transaction1.Data.Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", transaction1.Data.Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", "2022-02", 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope2Month v3.MonthConfigResponse
	test.DecodeResponse(suite.T(), &recorder, &envelope2Month)

	// No spend on this envelope in January, therefore no allocation in february
	suite.Assert().True(envelope2Month.Data.Allocation.Equal(decimal.NewFromFloat(0)), "Expected: 0, got %s, Request ID: %s", envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestMonthsPostFails() {
	budgetAllocationsLink := suite.createTestBudget(suite.T(), v3.BudgetEditable{}).Data.Links.Month

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
			recorder := test.Request(t, http.MethodPost, tt.url, tt.body)
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestMonthsSorting verifies that categories and months are sorted correctly
func (suite *TestSuiteStandard) TestMonthsSorting() {
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})
	categoryU := suite.createTestCategory(suite.T(), v3.CategoryEditable{BudgetID: budget.Data.ID, Name: "Upkeep"})
	envelopeU := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: categoryU.Data.ID, Name: "Utilities"})
	envelopeM := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: categoryU.Data.ID, Name: "Muppets"})

	categoryA := suite.createTestCategory(suite.T(), v3.CategoryEditable{BudgetID: budget.Data.ID, Name: "Alphabetically first"})
	envelopeB := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: categoryA.Data.ID, Name: "Batteries"})
	envelopeC := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: categoryA.Data.ID, Name: "Chargers"})

	// Get month data
	recorder := test.Request(suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", types.MonthOf(time.Now()).String(), 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	// Parse month data
	var response v3.MonthResponse
	test.DecodeResponse(suite.T(), &recorder, &response)
	month := response.Data

	assert.Equal(suite.T(), categoryU.Data.ID, month.Categories[1].ID)
	assert.Equal(suite.T(), envelopeU.Data.ID, month.Categories[1].Envelopes[1].ID)
	assert.Equal(suite.T(), envelopeM.Data.ID, month.Categories[1].Envelopes[0].ID)

	assert.Equal(suite.T(), categoryA.Data.ID, month.Categories[0].ID)
	assert.Equal(suite.T(), envelopeB.Data.ID, month.Categories[0].Envelopes[0].ID)
	assert.Equal(suite.T(), envelopeC.Data.ID, month.Categories[0].Envelopes[1].ID)
}

// TestMonths verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestMonths() {
	budget := suite.createTestBudget(suite.T(), v3.BudgetEditable{})
	category := suite.createTestCategory(suite.T(), v3.CategoryEditable{BudgetID: budget.Data.ID, Name: "Upkeep"})
	envelope := suite.createTestEnvelope(suite.T(), v3.EnvelopeEditable{CategoryID: category.Data.ID, Name: "Utilities"})
	account := suite.createTestAccount(suite.T(), models.Account{BudgetID: budget.Data.ID, OnBudget: true, Name: "TestMonth"})
	externalAccount := suite.createTestAccount(suite.T(), models.Account{BudgetID: budget.Data.ID, External: true})

	// Allocate funds to the months
	allocations := []struct {
		month  types.Month
		amount decimal.Decimal
	}{
		{types.NewMonth(2022, 1), decimal.NewFromFloat(20.99)},
		{types.NewMonth(2022, 2), decimal.NewFromFloat(47.12)},
		{types.NewMonth(2022, 3), decimal.NewFromFloat(31.17)},
	}

	for _, allocation := range allocations {
		suite.patchTestMonthConfig(suite.T(), envelope.Data.ID, allocation.month, models.MonthConfigCreate{
			Allocation: allocation.amount,
		})
	}

	_ = suite.createTestTransaction(suite.T(), models.Transaction{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
	})

	_ = suite.createTestTransaction(suite.T(), models.Transaction{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
	})

	_ = suite.createTestTransaction(suite.T(), models.Transaction{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
	})

	_ = suite.createTestTransaction(suite.T(), models.Transaction{
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
		result v3.Month
	}{
		{
			types.NewMonth(2022, 1),
			v3.Month{
				Month:      types.NewMonth(2022, 1),
				Income:     decimal.NewFromFloat(0),
				Balance:    decimal.NewFromFloat(10.99),
				Spent:      decimal.NewFromFloat(-10),
				Allocation: decimal.NewFromFloat(20.99),
				Available:  decimal.NewFromFloat(-20.99),
				Categories: []v3.CategoryEnvelopes{
					{
						Category:   *category.Data,
						Balance:    decimal.NewFromFloat(10.99),
						Spent:      decimal.NewFromFloat(-10),
						Allocation: decimal.NewFromFloat(20.99),
						Envelopes: []v3.EnvelopeMonth{
							{
								Envelope:   *envelope.Data,
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
			v3.Month{
				Month:      types.NewMonth(2022, 2),
				Income:     decimal.NewFromFloat(0),
				Balance:    decimal.NewFromFloat(53.11),
				Spent:      decimal.NewFromFloat(-5),
				Allocation: decimal.NewFromFloat(47.12),
				Available:  decimal.NewFromFloat(-68.11),
				Categories: []v3.CategoryEnvelopes{
					{
						Category:   *category.Data,
						Balance:    decimal.NewFromFloat(53.11),
						Spent:      decimal.NewFromFloat(-5),
						Allocation: decimal.NewFromFloat(47.12),
						Envelopes: []v3.EnvelopeMonth{
							{
								Envelope:   *envelope.Data,
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
			v3.Month{
				Month:      types.NewMonth(2022, 3),
				Income:     decimal.NewFromFloat(1500),
				Balance:    decimal.NewFromFloat(69.28),
				Spent:      decimal.NewFromFloat(-15),
				Allocation: decimal.NewFromFloat(31.17),
				Available:  decimal.NewFromFloat(1400.72),
				Categories: []v3.CategoryEnvelopes{
					{
						Category:   *category.Data,
						Balance:    decimal.NewFromFloat(69.28),
						Spent:      decimal.NewFromFloat(-15),
						Allocation: decimal.NewFromFloat(31.17),
						Envelopes: []v3.EnvelopeMonth{
							{
								Envelope:   *envelope.Data,
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
			recorder := test.Request(suite.T(), http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", tt.month.String(), 1), "")
			test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)

			// Parse month data
			var response v3.MonthResponse
			test.DecodeResponse(t, &recorder, &response)
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
