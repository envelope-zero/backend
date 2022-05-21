package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetBudgets(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets", "")

	var response controllers.BudgetListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 1) {
		assert.FailNow(t, "Response does not have exactly 1 item")
	}

	assert.Equal(t, "Testing Budget", response.Data[0].Name)
	assert.Equal(t, "GNU: Terry Pratchett", response.Data[0].Note)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)
}

func TestNoBudgetNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestBudgetInvalidIDs(t *testing.T) {
	/*
	 *  GET
	 */
	r := test.Request(t, http.MethodGet, "/v1/budgets/-56", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/budgets/notANumber", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/budgets/23", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/budgets/d19a622f-broken-uuid/2022-01", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(t, http.MethodPatch, "/v1/budgets/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodPatch, "/v1/budgets/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(t, http.MethodDelete, "/v1/budgets/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodDelete, "/v1/budgets/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budgetObject, savedObject controllers.BudgetResponse
	test.DecodeResponse(t, &recorder, &budgetObject)

	recorder = test.Request(t, "GET", budgetObject.Data.Links.Self, "")
	test.DecodeResponse(t, &recorder, &savedObject)

	assert.Equal(t, savedObject, budgetObject)
}

func TestCreateBrokenBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "createdAt": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateBudgetNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func TestBudgetMonth(t *testing.T) {
	var budgetList controllers.BudgetListResponse
	r := test.Request(t, http.MethodGet, "/v1/budgets", "")
	test.DecodeResponse(t, &r, &budgetList)

	var budgetMonth controllers.BudgetMonthResponse

	tests := []struct {
		path     string
		response controllers.BudgetMonthResponse
	}{
		{
			fmt.Sprintf("/v1/budgets/%s/2022-01", budgetList.Data[0].ID),
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
			fmt.Sprintf("/v1/budgets/%s/2022-02", budgetList.Data[0].ID),
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
			fmt.Sprintf("/v1/budgets/%s/2022-03", budgetList.Data[0].ID),
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

	for _, tt := range tests {
		r := test.Request(t, "GET", tt.path, "")
		test.AssertHTTPStatus(t, http.StatusOK, &r)
		test.DecodeResponse(t, &r, &budgetMonth)

		if !assert.Len(t, budgetMonth.Data.Envelopes, len(tt.response.Data.Envelopes)) {
			assert.FailNow(t, "Response length does not match!", "Response does not have exactly %v item(s)", len(tt.response.Data.Envelopes))
		}

		for i, envelope := range budgetMonth.Data.Envelopes {
			assert.True(t, envelope.Spent.Equal(tt.response.Data.Envelopes[i].Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[i].Spent, envelope.Spent, budgetMonth.Data)
			assert.True(t, envelope.Balance.Equal(tt.response.Data.Envelopes[i].Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[i].Balance, envelope.Balance, budgetMonth.Data)
			assert.True(t, envelope.Allocation.Equal(tt.response.Data.Envelopes[i].Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[i].Allocation, envelope.Allocation, budgetMonth.Data)
		}
	}
}

// TestBudgetMonthNonExistent verifies that month requests for non-existing budgets return a HTTP 404 Not Found.
func TestBudgetMonthNonExistent(t *testing.T) {
	r := test.Request(t, "GET", "/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd/2022-01", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &r)
}

// TestBudgetMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func TestBudgetMonthZero(t *testing.T) {
	var budgetList controllers.BudgetListResponse
	r := test.Request(t, http.MethodGet, "/v1/budgets", "")
	test.DecodeResponse(t, &r, &budgetList)

	r = test.Request(t, "GET", fmt.Sprintf("/v1/budgets/%s/0001-01", budgetList.Data[0].ID), "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

// TestBudgetMonthInvalid tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func TestBudgetMonthInvalid(t *testing.T) {
	var budgetList controllers.BudgetListResponse
	r := test.Request(t, http.MethodGet, "/v1/budgets", "")
	test.DecodeResponse(t, &r, &budgetList)

	r = test.Request(t, "GET", fmt.Sprintf("/v1/budgets/%s/December-2020", budgetList.Data[0].ID), "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestUpdateBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(t, &recorder, &budget)

	recorder = test.Request(t, "PATCH", budget.Data.Links.Self, `{ "name": "Updated new budget" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedBudget controllers.BudgetResponse
	test.DecodeResponse(t, &recorder, &updatedBudget)

	assert.Equal(t, budget.Data.Note, updatedBudget.Data.Note)
	assert.Equal(t, "Updated new budget", updatedBudget.Data.Name)
}

func TestUpdateBudgetBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(t, &recorder, &budget)

	recorder = test.Request(t, "PATCH", budget.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingBudget(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/a29bd123-beec-47de-a9cd-b6f7483fe00f", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(t, &recorder, &budget)

	recorder = test.Request(t, "DELETE", budget.Data.Links.Self, "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingBudget(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/c3d34346-609a-4734-9364-98f5b0100150", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteBudgetWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(t, &recorder, &budget)

	recorder = test.Request(t, "DELETE", budget.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
