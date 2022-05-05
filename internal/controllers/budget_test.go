package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type BudgetListResponse struct {
	test.APIResponse
	Data []models.Budget
}

type BudgetDetailResponse struct {
	test.APIResponse
	Data models.Budget
}
type BudgetMonthResponse struct {
	test.APIResponse
	Data models.BudgetMonth
}

func TestGetBudgets(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets", "")

	var response BudgetListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 1) {
		assert.FailNow(t, "Response does not have exactly 1 item")
	}

	assert.Equal(t, "Testing Budget", response.Data[0].Name)
	assert.Equal(t, "GNU: Terry Pratchett", response.Data[0].Note)

	diff := time.Now().Sub(response.Data[0].CreatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)

	diff = time.Now().Sub(response.Data[0].UpdatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)
}

func TestNoBudgetNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/2", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

// TestBudgetInvalidIDs verifies that on non-number requests for budget IDs,
// the API returs a Bad Request status code.
func TestBudgetInvalidIDs(t *testing.T) {
	r := test.Request(t, "GET", "/v1/budgets/-17", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "GET", "/v1/budgets/DefinitelyNotAUint64", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiBudget BudgetDetailResponse
	test.DecodeResponse(t, &recorder, &apiBudget)

	var dbBudget models.Budget
	models.DB.First(&dbBudget, apiBudget.Data.ID)

	assert.Equal(t, dbBudget, apiBudget.Data)
}

func TestCreateBrokenBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "createdAt": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateBudgetNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetBudget(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1", "")
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var budget BudgetDetailResponse
	test.DecodeResponse(t, &recorder, &budget)

	var dbBudget models.Budget
	models.DB.First(&dbBudget, budget.Data.ID)

	assert.Equal(t, dbBudget, budget.Data)
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func TestBudgetMonth(t *testing.T) {
	var budgetMonth BudgetMonthResponse

	tests := []struct {
		path     string
		response BudgetMonthResponse
	}{
		{
			"/v1/budgets/1/2022-01",
			BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							ID:         1,
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
			"/v1/budgets/1/2022-02",
			BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month: time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							ID:         1,
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
			"/v1/budgets/1/2022-03",
			BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month: time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							ID:         1,
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

	for _, tt := range tests {
		r := test.Request(t, "GET", tt.path, "")
		test.AssertHTTPStatus(t, http.StatusOK, &r)
		test.DecodeResponse(t, &r, &budgetMonth)

		if !assert.Len(t, budgetMonth.Data.Envelopes, len(tt.response.Data.Envelopes)) {
			assert.FailNow(t, "Reponse length does not match!", "Response does not have exactly %v item(s)", len(tt.response.Data.Envelopes))
		}

		for i, envelope := range budgetMonth.Data.Envelopes {
			assert.True(t, envelope.Spent.Equal(tt.response.Data.Envelopes[i].Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[i].Spent, envelope.Spent, budgetMonth.Data)
			assert.True(t, envelope.Balance.Equal(tt.response.Data.Envelopes[i].Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[i].Balance, envelope.Balance, budgetMonth.Data)
			assert.True(t, envelope.Allocation.Equal(tt.response.Data.Envelopes[i].Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[i].Allocation, envelope.Allocation, budgetMonth.Data)
		}
	}
}

// TestBudgetMonthInvalid verifies that non-parseable requests return a HTTP 400 Bad Request.
func TestBudgetMonthInvalid(t *testing.T) {
	r := test.Request(t, "GET", "/v1/budgets/1/Stonks!", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

// TestBudgetMonthNonExistent verifies that month requests for non-existing budgets return a HTTP 404 Not Found.
func TestBudgetMonthNonExistent(t *testing.T) {
	r := test.Request(t, "GET", "/v1/budgets/831/2022-01", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &r)
}

// TestBudgetMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func TestBudgetMonthZero(t *testing.T) {
	r := test.Request(t, "GET", "/v1/budgets/1/0001-01", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestUpdateBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget BudgetDetailResponse
	test.DecodeResponse(t, &recorder, &budget)

	path := fmt.Sprintf("/v1/budgets/%v", budget.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new budget" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedBudget BudgetDetailResponse
	test.DecodeResponse(t, &recorder, &updatedBudget)

	assert.Equal(t, budget.Data.Note, updatedBudget.Data.Note)
	assert.Equal(t, "Updated new budget", updatedBudget.Data.Name)
}

func TestUpdateBudgetBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget BudgetDetailResponse
	test.DecodeResponse(t, &recorder, &budget)

	path := fmt.Sprintf("/v1/budgets/%v", budget.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingBudget(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/48902805", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget BudgetDetailResponse
	test.DecodeResponse(t, &recorder, &budget)

	path := fmt.Sprintf("/v1/budgets/%v", budget.Data.ID)
	recorder = test.Request(t, "DELETE", path, "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingBudget(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteBudgetWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget BudgetDetailResponse
	test.DecodeResponse(t, &recorder, &budget)

	path := fmt.Sprintf("/v1/budgets/%v", budget.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
