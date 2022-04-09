package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
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

func TestGetBudgets(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets", "")

	var response BudgetListResponse
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

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

func TestCreateBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiBudget BudgetDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiBudget)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

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
	err := json.NewDecoder(recorder.Body).Decode(&budget)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbBudget models.Budget
	models.DB.First(&dbBudget, budget.Data.ID)

	assert.Equal(t, dbBudget, budget.Data)
}

func TestUpdateBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget BudgetDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&budget)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/%v", budget.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new budget" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedBudget BudgetDetailResponse
	err = json.NewDecoder(recorder.Body).Decode(&updatedBudget)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, budget.Data.Note, updatedBudget.Data.Note)
	assert.Equal(t, "Updated new budget", updatedBudget.Data.Name)
}

func TestUpdateBudgetBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var budget BudgetDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&budget)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

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
	err := json.NewDecoder(recorder.Body).Decode(&budget)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

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
	err := json.NewDecoder(recorder.Body).Decode(&budget)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/%v", budget.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
