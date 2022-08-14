package controllers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func createTestBudget(t *testing.T, c models.BudgetCreate) controllers.BudgetResponse {
	r := test.Request(t, http.MethodPost, "http://example.com/v1/budgets", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var a controllers.BudgetResponse
	test.DecodeResponse(t, &r, &a)

	return a
}

func (suite *TestSuiteEnv) TestOptionsBudget() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v1/budgets", uuid.New())
	recorder := test.Request(suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	recorder = test.Request(suite.T(), http.MethodOptions, "http://example.com/v1/budgets/NotParseableAsUUID", "")
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.Self
	recorder = test.Request(suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteEnv) TestOptionsBudgetMonth() {
	budgetLink := createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.Month

	recorder := test.Request(suite.T(), http.MethodOptions, strings.Replace(budgetLink, "YYYY-MM", "1970-01", 1), "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
	assert.Equal(suite.T(), recorder.Header().Get("allow"), "GET")

	// Bad Request for invalid UUID
	recorder = test.Request(suite.T(), http.MethodOptions, "http://example.com/v1/budgets/nouuid/2022-01", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)

	// Bad Request for invalid month
	recorder = test.Request(suite.T(), http.MethodOptions, budgetLink, "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)

	// Not found for non-existing budget
	recorder = test.Request(suite.T(), http.MethodOptions, "http://example.com/v1/budgets/5b95e1a9-522d-4a36-9074-32f7c2ff0513/1980-06", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestOptionsBudgetMonthAllocations() {
	budgetAllocationsLink := createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.MonthAllocations

	recorder := test.Request(suite.T(), http.MethodOptions, strings.Replace(budgetAllocationsLink, "YYYY-MM", "1970-01", 1), "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
	assert.Equal(suite.T(), recorder.Header().Get("allow"), "DELETE")

	// Bad Request for invalid UUID
	recorder = test.Request(suite.T(), http.MethodOptions, "http://example.com/v1/budgets/nouuid/2022-01/allocations", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)

	// Bad Request for invalid months
	recorder = test.Request(suite.T(), http.MethodOptions, budgetAllocationsLink, "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)

	// Not found for non-existing budget
	recorder = test.Request(suite.T(), http.MethodOptions, "http://example.com/v1/budgets/059cdead-249f-4f94-8d29-16a80c6b4a09/2032-03/allocations", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestGetBudgets() {
	_ = createTestBudget(suite.T(), models.BudgetCreate{})

	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets", "")

	var response controllers.BudgetListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)
}

func (suite *TestSuiteEnv) TestGetBudgetsFilter() {
	_ = createTestBudget(suite.T(), models.BudgetCreate{
		Name:     "Exact String Match",
		Note:     "This is a specific note",
		Currency: "",
	})

	_ = createTestBudget(suite.T(), models.BudgetCreate{
		Name:     "",
		Note:     "This is a specific note",
		Currency: "$",
	})

	_ = createTestBudget(suite.T(), models.BudgetCreate{
		Name:     "Another String",
		Note:     "A different note",
		Currency: "€",
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Currency: €", "currency=€", 1},
		{"Currency: $", "currency=$", 1},
		{"Currency & Name", "currency=€&name=Another String", 1},
		{"Note", "note=This is a specific note", 2},
		{"Name", "name=Exact String Match", 1},
		{"Empty Name with Note", "name=&note=This is a specific note", 1},
		{"No currency", "currency=", 1},
		{"No name", "name=", 1},
	}

	var re controllers.BudgetListResponse
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v1/budgets?%s", tt.query), "")
			test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
			test.DecodeResponse(suite.T(), &r, &re)
			assert.Equal(suite.T(), tt.len, len(re.Data))
		})
	}
}

func (suite *TestSuiteEnv) TestNoBudgetNotFound() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestBudgetInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/d19a622f-broken-uuid/2022-01", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/budgets/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/budgets/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/budgets/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/budgets/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateBudget() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budgetObject, savedObject controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budgetObject)

	recorder = test.Request(suite.T(), http.MethodGet, budgetObject.Data.Links.Self, "")
	test.DecodeResponse(suite.T(), &recorder, &savedObject)

	assert.Equal(suite.T(), savedObject, budgetObject)
}

func (suite *TestSuiteEnv) TestCreateBrokenBudget() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "createdAt": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateBudgetNoBody() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteEnv) TestBudgetMonth() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})
	category := createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	account := createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID})
	externalAccount := createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID, External: true})

	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(20.99),
	})

	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(47.12),
	})

	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(31.17),
	})

	_ = createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	tests := []struct {
		path     string
		response controllers.BudgetMonthResponse
	}{
		{
			fmt.Sprintf("%s/2022-01", budget.Data.Links.Self),
			controllers.BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
							Spent:      decimal.NewFromFloat(-10),
							Balance:    decimal.NewFromFloat(10.99),
							Allocation: decimal.NewFromFloat(20.99),
						},
					},
				},
			},
		},
		{
			fmt.Sprintf("%s/2022-02", budget.Data.Links.Self),
			controllers.BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month: time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
							Spent:      decimal.NewFromFloat(-5),
							Balance:    decimal.NewFromFloat(42.12),
							Allocation: decimal.NewFromFloat(47.12),
						},
					},
				},
			},
		},
		{
			fmt.Sprintf("%s/2022-03", budget.Data.Links.Self),
			controllers.BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month: time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
							Spent:      decimal.NewFromFloat(-15),
							Balance:    decimal.NewFromFloat(16.17),
							Allocation: decimal.NewFromFloat(31.17),
						},
					},
				},
			},
		},
	}

	var budgetMonth controllers.BudgetMonthResponse
	for _, tt := range tests {
		r := test.Request(suite.T(), http.MethodGet, tt.path, "")
		test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
		test.DecodeResponse(suite.T(), &r, &budgetMonth)

		// assert.FailNow(suite.T(), "BudgetMonth", budgetMonth)

		if !assert.Len(suite.T(), budgetMonth.Data.Envelopes, 1) {
			assert.FailNow(suite.T(), "Response length does not match!", "Response does not have exactly 1 item")
		}

		envelope := budgetMonth.Data.Envelopes[0]
		assert.True(suite.T(), envelope.Spent.Equal(tt.response.Data.Envelopes[0].Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[0].Spent, envelope.Spent, budgetMonth.Data)
		assert.True(suite.T(), envelope.Balance.Equal(tt.response.Data.Envelopes[0].Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[0].Balance, envelope.Balance, budgetMonth.Data)
		assert.True(suite.T(), envelope.Allocation.Equal(tt.response.Data.Envelopes[0].Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[0].Allocation, envelope.Allocation, budgetMonth.Data)
	}
}

func (suite *TestSuiteEnv) TestBudgetMonthBudgeted() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})
	category := createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	envelopeZero := createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Zero"})

	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelopeZero.Data.ID,
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(19.01),
	})

	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(20.99),
	})

	var budgetMonth controllers.BudgetMonthResponse

	r := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("%s/2022-01", budget.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
	test.DecodeResponse(suite.T(), &r, &budgetMonth)

	assert.True(suite.T(), budgetMonth.Data.Budgeted.Equal(decimal.NewFromFloat(40)), "Calculation of budgeted sum for a month is off. Should be 40, is %s", budgetMonth.Data.Budgeted)
}

// TestBudgetMonthNonExistent verifies that month requests for non-existing budgets return a HTTP 404 Not Found.
func (suite *TestSuiteEnv) TestBudgetMonthNonExistent() {
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd/2022-01", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &r)
}

// TestBudgetMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteEnv) TestBudgetMonthZero() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})

	r := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("%s/0001-01", budget.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

// TestBudgetMonthInvalid tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteEnv) TestBudgetMonthInvalid() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})

	recorder := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("%s/December-2020", budget.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateBudget() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), http.MethodPatch, budget.Data.Links.Self, map[string]any{
		"name": "Updated new budget",
		"note": "",
	})
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedBudget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedBudget)

	assert.Equal(suite.T(), "", updatedBudget.Data.Note)
	assert.Equal(suite.T(), "Updated new budget", updatedBudget.Data.Name)
}

func (suite *TestSuiteEnv) TestUpdateBudgetBrokenJSON() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), http.MethodPatch, budget.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateBudgetInvalidType() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), http.MethodPatch, budget.Data.Links.Self, map[string]any{
		"name": 2,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingBudget() {
	recorder := test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/budgets/a29bd123-beec-47de-a9cd-b6f7483fe00f", `{ "name": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteBudget() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), http.MethodDelete, budget.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingBudget() {
	recorder := test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/budgets/c3d34346-609a-4734-9364-98f5b0100150", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteBudgetWithBody() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), http.MethodDelete, budget.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteAllocationsMonth() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})
	category := createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})

	allocation1 := createTestAllocation(suite.T(), models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := createTestAllocation(suite.T(), models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope2.Data.ID,
	})

	// Clear allocations
	recorder := test.Request(suite.T(), http.MethodDelete, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)

	// Verify that allocations are deleted
	recorder = test.Request(suite.T(), http.MethodGet, allocation1.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)

	recorder = test.Request(suite.T(), http.MethodGet, allocation2.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteAllocationsMonthFailures() {
	budgetAllocationsLink := createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.MonthAllocations

	// Bad Request for invalid UUID
	recorder := test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/budgets/nouuid/2022-01/allocations", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)

	// Bad Request for invalid months
	recorder = test.Request(suite.T(), http.MethodDelete, budgetAllocationsLink, "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)

	// Not found for non-existing budget
	recorder = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/budgets/059cdead-249f-4f94-8d29-16a80c6b4a09/2032-03/allocations", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}
