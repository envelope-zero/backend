package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type AllocationListResponse struct {
	test.APIResponse
	Data []models.Allocation
}

type AllocationDetailResponse struct {
	test.APIResponse
	Data models.Allocation
}

func TestGetAllocations(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/1/allocations", "")

	var response AllocationListResponse
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 3) {
		assert.FailNow(t, "Response does not have exactly 3 items")
	}

	assert.Equal(t, uint64(1), response.Data[0].EnvelopeID)
	assert.Equal(t, uint8(1), response.Data[0].Month)
	assert.Equal(t, uint(2022), response.Data[0].Year)

	if !decimal.NewFromFloat(20.99).Equal(response.Data[0].Amount) {
		assert.Fail(t, "Allocation amount does not equal 20.99", response.Data[0].Amount)
	}

	diff := time.Now().Sub(response.Data[0].CreatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)

	diff = time.Now().Sub(response.Data[0].UpdatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)
}

func TestNoAllocationNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/1/allocations/60", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes/1/allocations", `{ "month": 10, "year": 2022, "amount": 15.42 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiAllocation AllocationDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiAllocation)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbAllocation models.Allocation
	models.DB.First(&dbAllocation, apiAllocation.Data.ID)

	if !decimal.NewFromFloat(15.42).Equal(apiAllocation.Data.Amount) {
		assert.Fail(t, "Allocation amount does not equal 15.42", apiAllocation.Data.Amount)
	}

	// Set the balance to 0 to compare to the database object
	apiAllocation.Data.Amount = decimal.NewFromFloat(0)
	dbAllocation.Amount = decimal.NewFromFloat(0)

	assert.Equal(t, dbAllocation, apiAllocation.Data)
}

func TestCreateBrokenAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes/1/allocations", `{ "createdAt": "New Allocation" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateDuplicateAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes/1/allocations", `{ "year": 2022, "month": 2 }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAllocationNoMonth(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes/1/allocations", `{ "year": 2022, "month": 17 }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAllocationNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes/1/allocations", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetAllocation(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/1/allocations/1", "")
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var allocation AllocationDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&allocation)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbAllocation models.Allocation
	models.DB.First(&dbAllocation, allocation.Data.ID)

	assert.Equal(t, dbAllocation, allocation.Data)
}

func TestUpdateAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes/1/allocations", `{ "year": 2100, "month": 6 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var allocation AllocationDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&allocation)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/1/envelopes/1/allocations/%v", allocation.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{  "year": 2022 }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedAllocation AllocationDetailResponse
	err = json.NewDecoder(recorder.Body).Decode(&updatedAllocation)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, uint(2022), updatedAllocation.Data.Year)
}

func TestUpdateAllocationBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes/1/allocations", `{ "year": 2017, "month": 11 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var allocation AllocationDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&allocation)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/1/envelopes/1/allocations/%v", allocation.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingAllocation(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/1/categories/1/envelopes/1/allocations/48902805", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAllocation(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/categories/1/envelopes/1/allocations/1", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingAllocation(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/categories/1/envelopes/1/allocations/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAllocationWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes/1/allocations", `{ "year": 2070, "month": 12}`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var allocation AllocationDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&allocation)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/1/envelopes/1/allocations/%v", allocation.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
