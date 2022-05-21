package controllers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func createTestAllocation(t *testing.T, c models.AllocationCreate) controllers.AllocationResponse {
	r := test.Request(t, "POST", "/v1/allocations", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var a controllers.AllocationResponse
	test.DecodeResponse(t, &r, &a)

	return a
}

func TestGetAllocations(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/allocations", "")

	var response controllers.AllocationListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 3) {
		assert.FailNow(t, "Response does not have exactly 3 items")
	}

	assert.Equal(t, uint8(1), response.Data[0].Month)
	assert.Equal(t, uint(2022), response.Data[0].Year)

	if !decimal.NewFromFloat(20.99).Equal(response.Data[0].Amount) {
		assert.Fail(t, "Allocation amount does not equal 20.99", response.Data[0].Amount)
	}

	assert.LessOrEqual(t, time.Since(response.Data[0].CreatedAt), test.TOLERANCE)
	assert.LessOrEqual(t, time.Since(response.Data[0].UpdatedAt), test.TOLERANCE)
}

func TestNoAllocationNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/allocations/f8b93ce2-309f-4e99-8886-6ab960df99c3", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestAllocationInvalidIDs(t *testing.T) {
	/*
	 *  GET
	 */
	r := test.Request(t, http.MethodGet, "/v1/allocations/-56", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/allocations/notANumber", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/allocations/23", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(t, http.MethodPatch, "/v1/allocations/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodPatch, "/v1/allocations/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(t, http.MethodDelete, "/v1/allocations/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodDelete, "/v1/allocations/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateAllocation(t *testing.T) {
	a := createTestAllocation(t, models.AllocationCreate{
		Month:  10,
		Year:   2022,
		Amount: decimal.NewFromFloat(15.42),
	})

	if !decimal.NewFromFloat(15.42).Equal(a.Data.Amount) {
		assert.Fail(t, "Allocation amount does not equal 15.42", a.Data.Amount)
	}
}

func TestCreateBrokenAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", `{ "createdAt": "New Allocation" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAllocationNonExistingEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", models.AllocationCreate{EnvelopeID: uuid.New()})
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateDuplicateAllocation(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", models.AllocationCreate{Year: 2022, Month: 2})
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAllocationNoMonth(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", models.AllocationCreate{Year: 2022, Month: 17})
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAllocationNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/allocations", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetAllocation(t *testing.T) {
	a := createTestAllocation(t, models.AllocationCreate{
		Year:  2022,
		Month: 8,
	})

	r := test.Request(t, http.MethodGet, a.Data.Links.Self, "")
	assert.Equal(t, http.StatusOK, r.Code)
}

func TestUpdateAllocation(t *testing.T) {
	a := createTestAllocation(t, models.AllocationCreate{Year: 2100, Month: 6})

	r := test.Request(t, "PATCH", a.Data.Links.Self, models.AllocationCreate{Year: 2022})
	test.AssertHTTPStatus(t, http.StatusOK, &r)

	var updatedAllocation controllers.AllocationResponse
	test.DecodeResponse(t, &r, &updatedAllocation)

	assert.Equal(t, uint(2022), updatedAllocation.Data.Year)
}

func TestUpdateAllocationBroken(t *testing.T) {
	a := createTestAllocation(t, models.AllocationCreate{Year: 2100, Month: 6})

	r := test.Request(t, "PATCH", a.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestUpdateNonExistingAllocation(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/allocations/df684988-31df-444c-8aaa-b53195d55d6e", models.AllocationCreate{Month: 2})
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAllocation(t *testing.T) {
	a := createTestAllocation(t, models.AllocationCreate{Year: 2033, Month: 11})
	r := test.Request(t, "DELETE", a.Data.Links.Self, "")

	test.AssertHTTPStatus(t, http.StatusNoContent, &r)
}

func TestDeleteNonExistingAllocation(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/allocations/34ac51a7-431c-454b-ba29-feaefeae70d5", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAllocationWithBody(t *testing.T) {
	a := createTestAllocation(t, models.AllocationCreate{Year: 2070, Month: 12})

	r := test.Request(t, "DELETE", a.Data.Links.Self, models.AllocationCreate{Year: 2011, Month: 3})
	test.AssertHTTPStatus(t, http.StatusNoContent, &r)
}
