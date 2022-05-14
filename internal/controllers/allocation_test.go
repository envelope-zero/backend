package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/controllers"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetAllocations(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/allocations", "")

	var response controllers.AllocationListResponse
	test.DecodeResponse(t, &recorder, &response)

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

	assert.LessOrEqual(t, time.Since(response.Data[0].CreatedAt), test.TOLERANCE)
	assert.LessOrEqual(t, time.Since(response.Data[0].UpdatedAt), test.TOLERANCE)
}

func TestNoAllocationNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/allocations/60", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

// TestAllocationInvalidIDs verifies that on non-number requests for allocation IDs,
// the API returs a Bad Request status code.
func TestAllocationInvalidIDs(t *testing.T) {
	r := test.Request(t, "GET", "/v1/allocations/-2", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "GET", "/v1/allocations/RoadWorkAhead", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "PATCH", "/v1/allocations/SneezingBecauseAllergies", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "DELETE", "/v1/allocations/;!", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "month": 10, "year": 2022, "amount": 15.42 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiAllocation controllers.AllocationResponse
	test.DecodeResponse(t, &recorder, &apiAllocation)

	var dbAllocation models.Allocation
	models.DB.First(&dbAllocation, apiAllocation.Data.ID)

	if !decimal.NewFromFloat(15.42).Equal(apiAllocation.Data.Amount) {
		assert.Fail(t, "Allocation amount does not equal 15.42", apiAllocation.Data.Amount)
	}
}

func TestCreateBrokenAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "createdAt": "New Allocation" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAllocationNonExistingEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "envelopeId": 2581 }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateDuplicateAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "year": 2022, "month": 2 }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAllocationNoMonth(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "year": 2022, "month": 17 }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAllocationNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetAllocation(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/allocations/1", "")
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var allocationObject, savedAllocation controllers.AllocationResponse
	test.DecodeResponse(t, &recorder, &allocationObject)

	recorder = test.Request(t, "GET", allocationObject.Data.Links.Self, "")
	test.DecodeResponse(t, &recorder, &savedAllocation)

	assert.Equal(t, savedAllocation, allocationObject)
}

func TestUpdateAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "year": 2100, "month": 6 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var allocation controllers.AllocationResponse
	test.DecodeResponse(t, &recorder, &allocation)

	path := fmt.Sprintf("/v1/allocations/%v", allocation.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{  "year": 2022 }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedAllocation controllers.AllocationResponse
	test.DecodeResponse(t, &recorder, &updatedAllocation)

	assert.Equal(t, uint(2022), updatedAllocation.Data.Year)
}

func TestUpdateAllocationBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "year": 2017, "month": 11 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var allocation controllers.AllocationResponse
	test.DecodeResponse(t, &recorder, &allocation)

	path := fmt.Sprintf("/v1/allocations/%v", allocation.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingAllocation(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/allocations/48902805", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "year": 2033, "month": 11 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var allocation controllers.AllocationResponse
	test.DecodeResponse(t, &recorder, &allocation)

	path := fmt.Sprintf("/v1/allocations/%v", allocation.Data.ID)
	recorder = test.Request(t, "DELETE", path, "")

	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingAllocation(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/allocations/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAllocationWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "year": 2070, "month": 12}`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var allocation controllers.AllocationResponse
	test.DecodeResponse(t, &recorder, &allocation)

	path := fmt.Sprintf("/v1/allocations/%v", allocation.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
