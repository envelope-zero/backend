package controllers_test

import (
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v2/internal/types"
	"github.com/envelope-zero/backend/v2/pkg/controllers"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/envelope-zero/backend/v2/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestAllocation(c models.AllocationCreate, expectedStatus ...int) controllers.AllocationResponse {
	if c.EnvelopeID == uuid.Nil {
		c.EnvelopeID = suite.createTestEnvelope(models.EnvelopeCreate{Name: "Transaction Test Envelope"}).Data.ID
	}

	// If no amount is set, set a random one
	if c.Amount.IsZero() {
		c.Amount = decimal.NewFromFloat(float64(rand.Intn(100000)) / 100.0)
	}

	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/allocations", c)
	suite.assertHTTPStatus(&r, expectedStatus...)

	var a controllers.AllocationResponse
	suite.decodeResponse(&r, &a)

	return a
}

func (suite *TestSuiteStandard) TestAllocations() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/allocations", "")
	suite.assertHTTPStatus(&recorder, http.StatusInternalServerError)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteStandard) TestOptionsAllocation() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v1/allocations", uuid.New())
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/allocations/NotParseableAsUUID", "")
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2022, 2)}).Data.Links.Self
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestGetAllocations() {
	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:  types.NewMonth(2022, 1),
		Amount: decimal.NewFromFloat(20.99),
	})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/allocations", "")

	var response controllers.AllocationListResponse
	suite.decodeResponse(&recorder, &response)

	suite.assertHTTPStatus(&recorder, http.StatusOK)
	assert.Len(suite.T(), response.Data, 1)
	assert.Equal(suite.T(), types.NewMonth(2022, 1), response.Data[0].Month)

	if !decimal.NewFromFloat(20.99).Equal(response.Data[0].Amount) {
		assert.Fail(suite.T(), "Allocation amount does not equal 20.99", response.Data[0].Amount)
	}

	assert.LessOrEqual(suite.T(), time.Since(response.Data[0].CreatedAt), tolerance)
	assert.LessOrEqual(suite.T(), time.Since(response.Data[0].UpdatedAt), tolerance)
}

func (suite *TestSuiteStandard) TestGetAllocationsFilter() {
	e1 := suite.createTestEnvelope(models.EnvelopeCreate{})
	e2 := suite.createTestEnvelope(models.EnvelopeCreate{})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: e1.Data.ID,
		Month:      types.NewMonth(2018, 9),
		Amount:     decimal.NewFromFloat(314.1592),
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: e1.Data.ID,
		Month:      types.NewMonth(2018, 10),
		Amount:     decimal.NewFromFloat(1371),
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: e2.Data.ID,
		Month:      types.NewMonth(2018, 9),
		Amount:     decimal.NewFromFloat(1204),
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Envelope 1", fmt.Sprintf("envelope=%s", e1.Data.ID), 2},
		{"Envelope Not Existing", "envelope=f1411c94-0ec6-417a-bb00-9e51d3c1c6e0", 0},
		{"Amount", "amount=1204", 1},
		{"Month", fmt.Sprintf("month=%s", types.NewMonth(2018, 9)), 2},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.AllocationListResponse
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v1/allocations?%s", tt.query), "")
			suite.assertHTTPStatus(&r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestNoAllocationNotFound() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/allocations/f8b93ce2-309f-4e99-8886-6ab960df99c3", "")

	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestGetAllocationsInvalidQuery() {
	tests := []string{
		"month=2022 Test Month",
		"amount=The cake is a lie",
		"envelope=NotAUUID",
	}

	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v1/allocations?%s", tt), "")
			suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
		})
	}
}

func (suite *TestSuiteStandard) TestAllocationInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/allocations/-56", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/allocations/notANumber", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/allocations/23", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	/*
	 * PATCH
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/allocations/-274", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/allocations/stringRandom", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	/*
	 * DELETE
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/allocations/-274", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/allocations/stringRandom", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateAllocation() {
	a := suite.createTestAllocation(models.AllocationCreate{
		Month:  types.NewMonth(2022, 10),
		Amount: decimal.NewFromFloat(15.42),
	})

	if !decimal.NewFromFloat(15.42).Equal(a.Data.Amount) {
		assert.Fail(suite.T(), "Allocation amount does not equal 15.42", a.Data.Amount)
	}
}

func (suite *TestSuiteStandard) TestCreateAllocationNoEnvelope() {
	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/allocations", models.Allocation{})
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateBrokenAllocation() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/allocations", `{ "createdAt": "New Allocation" }`)
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateAllocationNonExistingEnvelope() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/allocations", models.AllocationCreate{EnvelopeID: uuid.New()})
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestCreateDuplicateAllocation() {
	allocation := suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2022, 2)})
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/allocations", models.AllocationCreate{
		EnvelopeID: allocation.Data.EnvelopeID,
		Month:      allocation.Data.Month,
	})

	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateNonDuplicateAllocationSameMonth() {
	e1 := suite.createTestEnvelope(models.EnvelopeCreate{})
	e2 := suite.createTestEnvelope(models.EnvelopeCreate{})

	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 2),
		EnvelopeID: e1.Data.ID,
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 2),
		EnvelopeID: e2.Data.ID,
	})
}

func (suite *TestSuiteStandard) TestCreateAllocationNoBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/allocations", "")
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestGetAllocation() {
	a := suite.createTestAllocation(models.AllocationCreate{
		Month: types.NewMonth(2022, 8),
	})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, a.Data.Links.Self, "")
	assert.Equal(suite.T(), http.StatusOK, r.Code)
}

func (suite *TestSuiteStandard) TestUpdateAllocation() {
	a := suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2100, 6)})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"month": types.NewMonth(2022, 6),
	})
	suite.assertHTTPStatus(&r, http.StatusOK)

	var updatedAllocation controllers.AllocationResponse
	suite.decodeResponse(&r, &updatedAllocation)

	assert.Equal(suite.T(), 2022, time.Time(updatedAllocation.Data.Month).Year())
}

func (suite *TestSuiteStandard) TestUpdateAllocationZeroValues() {
	a := suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2100, 8)})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"month": types.NewMonth(0, 8),
	})
	suite.assertHTTPStatus(&r, http.StatusOK)

	var updatedAllocation controllers.AllocationResponse
	suite.decodeResponse(&r, &updatedAllocation)

	assert.Equal(suite.T(), 0, time.Time(updatedAllocation.Data.Month).Year(), "Year is not updated correctly")
}

func (suite *TestSuiteStandard) TestUpdateAllocationBrokenJSON() {
	a := suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2054, 5)})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, `{ "name": 2" }`)
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateAllocationInvalidType() {
	a := suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2062, 3)})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"month": "A long time ago in a galaxy far, far away",
	})
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateAllocationInvalidEnvelopeID() {
	a := suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2099, 11)})

	// Sets the EnvelopeID to uuid.Nil by not specifying it
	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, models.AllocationCreate{Month: types.NewMonth(2099, 11)})
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateNonExistingAllocation() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/allocations/df684988-31df-444c-8aaa-b53195d55d6e", models.AllocationCreate{Month: types.NewMonth(2142, 3)})
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteAllocation() {
	e := suite.createTestEnvelope(models.EnvelopeCreate{})
	a := suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2058, 7), EnvelopeID: e.Data.ID})
	r := test.Request(suite.controller, suite.T(), http.MethodDelete, a.Data.Links.Self, "")

	suite.assertHTTPStatus(&r, http.StatusNoContent)

	// Regression Test: Verify that allocations are hard deleted instantly to avoid problems
	// with the UNIQUE(id,month)
	_ = suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2058, 7), EnvelopeID: e.Data.ID})
}

func (suite *TestSuiteStandard) TestDeleteNonExistingAllocation() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/allocations/34ac51a7-431c-454b-ba29-feaefeae70d5", "")
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteAllocationWithBody() {
	a := suite.createTestAllocation(models.AllocationCreate{Month: types.NewMonth(2067, 3)})

	r := test.Request(suite.controller, suite.T(), http.MethodDelete, a.Data.Links.Self, models.AllocationCreate{Month: types.NewMonth(2067, 3)})
	suite.assertHTTPStatus(&r, http.StatusNoContent)
}

func (suite *TestSuiteStandard) TestDeleteNullAllocation() {
	r := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/allocations/00000000-0000-0000-0000-000000000000", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}
