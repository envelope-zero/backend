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
	if c.EnvelopeID == uuid.Nil {
		c.EnvelopeID = createTestEnvelope(t, models.EnvelopeCreate{Name: "Transaction Test Envelope"}).Data.ID
	}

	r := test.Request(t, http.MethodPost, "http://example.com/v1/allocations", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var a controllers.AllocationResponse
	test.DecodeResponse(t, &r, &a)

	return a
}

func (suite *TestSuiteEnv) TestGetAllocations() {
	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		Month:  1,
		Year:   2022,
		Amount: decimal.NewFromFloat(20.99),
	})

	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/allocations", "")

	var response controllers.AllocationListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)
	assert.Len(suite.T(), response.Data, 1)

	assert.Equal(suite.T(), uint8(1), response.Data[0].Month)
	assert.Equal(suite.T(), uint(2022), response.Data[0].Year)

	if !decimal.NewFromFloat(20.99).Equal(response.Data[0].Amount) {
		assert.Fail(suite.T(), "Allocation amount does not equal 20.99", response.Data[0].Amount)
	}

	assert.LessOrEqual(suite.T(), time.Since(response.Data[0].CreatedAt), test.TOLERANCE)
	assert.LessOrEqual(suite.T(), time.Since(response.Data[0].UpdatedAt), test.TOLERANCE)
}

func (suite *TestSuiteEnv) TestNoAllocationNotFound() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/allocations/f8b93ce2-309f-4e99-8886-6ab960df99c3", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestAllocationInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/allocations/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/allocations/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/allocations/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/allocations/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/allocations/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/allocations/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/allocations/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateAllocation() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{
		Month:  10,
		Year:   2022,
		Amount: decimal.NewFromFloat(15.42),
	})

	if !decimal.NewFromFloat(15.42).Equal(a.Data.Amount) {
		assert.Fail(suite.T(), "Allocation amount does not equal 15.42", a.Data.Amount)
	}
}

func (suite *TestSuiteEnv) TestCreateAllocationNoEnvelope() {
	r := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/allocations", models.Allocation{})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateBrokenAllocation() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/allocations", `{ "createdAt": "New Allocation" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateAllocationNonExistingEnvelope() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/allocations", models.AllocationCreate{EnvelopeID: uuid.New()})
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestCreateDuplicateAllocation() {
	allocation := createTestAllocation(suite.T(), models.AllocationCreate{Year: 2022, Month: 2})
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/allocations", models.AllocationCreate{
		EnvelopeID: allocation.Data.EnvelopeID,
		Month:      allocation.Data.Month,
		Year:       allocation.Data.Year,
	})

	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateAllocationNoMonth() {
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{})
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/allocations", models.AllocationCreate{EnvelopeID: envelope.Data.ID, Year: 2022})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateAllocationNoBody() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/allocations", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestGetAllocation() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{
		Year:  2022,
		Month: 8,
	})

	r := test.Request(suite.T(), http.MethodGet, a.Data.Links.Self, "")
	assert.Equal(suite.T(), http.StatusOK, r.Code)
}

func (suite *TestSuiteEnv) TestUpdateAllocation() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{Year: 2100, Month: 6})

	r := test.Request(suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"year": 2022,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)

	var updatedAllocation controllers.AllocationResponse
	test.DecodeResponse(suite.T(), &r, &updatedAllocation)

	assert.Equal(suite.T(), uint(2022), updatedAllocation.Data.Year)
}

func (suite *TestSuiteEnv) TestUpdateAllocationZeroValues() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{Year: 2100, Month: 6})

	r := test.Request(suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"year":  0,
		"month": 8,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)

	var updatedAllocation controllers.AllocationResponse
	test.DecodeResponse(suite.T(), &r, &updatedAllocation)

	assert.Equal(suite.T(), uint(0), updatedAllocation.Data.Year, "Year is not updated correctly")
}

func (suite *TestSuiteEnv) TestUpdateAllocationBrokenJSON() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{Year: 2100, Month: 6})

	r := test.Request(suite.T(), http.MethodPatch, a.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestUpdateAllocationInvalidType() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{Year: 2100, Month: 6})

	r := test.Request(suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"year": "A long time ago in a galaxy far, far away",
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestUpdateAllocationInvalidCategoryID() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{Year: 2100, Month: 6})

	// Sets the EnvelopeID to uuid.Nil
	r := test.Request(suite.T(), http.MethodPatch, a.Data.Links.Self, models.AllocationCreate{Month: 6, Year: 2100})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingAllocation() {
	recorder := test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/allocations/df684988-31df-444c-8aaa-b53195d55d6e", models.AllocationCreate{Month: 2})
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteAllocation() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{Year: 2033, Month: 11})
	r := test.Request(suite.T(), http.MethodDelete, a.Data.Links.Self, "")

	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &r)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingAllocation() {
	recorder := test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/allocations/34ac51a7-431c-454b-ba29-feaefeae70d5", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteAllocationWithBody() {
	a := createTestAllocation(suite.T(), models.AllocationCreate{Year: 2070, Month: 12})

	r := test.Request(suite.T(), http.MethodDelete, a.Data.Links.Self, models.AllocationCreate{Year: 2011, Month: 3})
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &r)
}

func (suite *TestSuiteEnv) TestDeleteNullAllocation() {
	r := test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/allocations/00000000-0000-0000-0000-000000000000", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}
