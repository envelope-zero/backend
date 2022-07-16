package controllers_test

import (
	"fmt"
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

func createTestEnvelope(t *testing.T, c models.EnvelopeCreate) controllers.EnvelopeResponse {
	if c.CategoryID == uuid.Nil {
		c.CategoryID = createTestCategory(t, models.CategoryCreate{Name: "Testing category"}).Data.ID
	}

	r := test.Request(t, "POST", "http://example.com/v1/envelopes", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var e controllers.EnvelopeResponse
	test.DecodeResponse(t, &r, &e)

	return e
}

func (suite *TestSuiteEnv) TestGetEnvelopes() {
	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/envelopes", "")

	var response controllers.EnvelopeListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	if !assert.Len(suite.T(), response.Data, 1) {
		assert.FailNow(suite.T(), "Response does not have exactly 1 item")
	}

	assert.Equal(suite.T(), "Utilities", response.Data[0].Name)
	assert.Equal(suite.T(), "Energy & Water", response.Data[0].Note)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)
}

func (suite *TestSuiteEnv) TestGetEnvelope() {
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{})
	recorder := test.Request(suite.T(), http.MethodGet, envelope.Data.Links.Self, "")

	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)
}

func (suite *TestSuiteEnv) TestNoEnvelopeNotFound() {
	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/envelopes/828f2483-dabd-4267-a223-e34b5f171978", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestEnvelopeInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/envelopes/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/envelopes/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/envelopes/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/envelopes/d19a622f-broken-uuid/2017-09", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/envelopes/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/envelopes/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/envelopes/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/envelopes/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateEnvelope() {
	_ = createTestEnvelope(suite.T(), models.EnvelopeCreate{Name: "New envelope", Note: "More tests something something"})
}

func (suite *TestSuiteEnv) TestCreateEnvelopeNoCategory() {
	r := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/envelopes", models.Envelope{})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateBrokenEnvelope() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/envelopes", `{ "createdAt": "New Envelope", "note": "More tests for envelopes to ensure less brokenness something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateEnvelopeNonExistingCategory() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/envelopes", `{ "categoryId": "5f0cd7b9-9788-4871-96f8-c816c9ae338a" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestCreateEnvelopeNoBody() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/envelopes", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

// TestEnvelopeMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteEnv) TestEnvelopeMonth() {
	var envelopeList controllers.EnvelopeListResponse
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/envelopes", "")
	test.DecodeResponse(suite.T(), &r, &envelopeList)

	var envelopeMonth controllers.EnvelopeMonthResponse

	tests := []struct {
		path          string
		envelopeMonth models.EnvelopeMonth
	}{
		{
			fmt.Sprintf("http://example.com/v1/envelopes/%s/2022-01", envelopeList.Data[0].ID),
			models.EnvelopeMonth{
				Name:       "Utilities",
				Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				Spent:      decimal.NewFromFloat(-10),
				Balance:    decimal.NewFromFloat(10.99),
				Allocation: decimal.NewFromFloat(20.99),
			},
		},
		{
			fmt.Sprintf("http://example.com/v1/envelopes/%s/2022-02", envelopeList.Data[0].ID),
			models.EnvelopeMonth{
				Name:       "Utilities",
				Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
				Spent:      decimal.NewFromFloat(-5),
				Balance:    decimal.NewFromFloat(42.12),
				Allocation: decimal.NewFromFloat(47.12),
			},
		},
		{
			fmt.Sprintf("http://example.com/v1/envelopes/%s/2022-03", envelopeList.Data[0].ID),
			models.EnvelopeMonth{
				Name:       "Utilities",
				Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
				Spent:      decimal.NewFromFloat(-15),
				Balance:    decimal.NewFromFloat(16.17),
				Allocation: decimal.NewFromFloat(31.17),
			},
		},
	}

	for _, tt := range tests {
		r := test.Request(suite.T(), "GET", tt.path, "")
		test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)

		test.DecodeResponse(suite.T(), &r, &envelopeMonth)
		assert.Equal(suite.T(), envelopeMonth.Data.Name, tt.envelopeMonth.Name)
		assert.Equal(suite.T(), envelopeMonth.Data.Month, tt.envelopeMonth.Month)
		assert.True(suite.T(), envelopeMonth.Data.Spent.Equal(tt.envelopeMonth.Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Spent, envelopeMonth.Data.Spent, envelopeMonth.Data)
		assert.True(suite.T(), envelopeMonth.Data.Balance.Equal(tt.envelopeMonth.Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Balance, envelopeMonth.Data.Balance, envelopeMonth.Data)
		assert.True(suite.T(), envelopeMonth.Data.Allocation.Equal(tt.envelopeMonth.Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Allocation, envelopeMonth.Data.Allocation, envelopeMonth.Data)
	}
}

func (suite *TestSuiteEnv) TestEnvelopeMonthInvalid() {
	var envelopeList controllers.EnvelopeListResponse
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/envelopes", "")
	test.DecodeResponse(suite.T(), &r, &envelopeList)

	// Test that non-parseable requests produce an error
	r = test.Request(suite.T(), "GET", fmt.Sprintf("http://example.com/v1/envelopes/%s/Stonks!", envelopeList.Data[0].ID), "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

// TestEnvelopeMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteEnv) TestEnvelopeMonthZero() {
	e := createTestEnvelope(suite.T(), models.EnvelopeCreate{})
	r := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("%s/0001-01", e.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestUpdateEnvelope() {
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{Name: "New envelope", Note: "Keks is a cuddly cat"})

	recorder := test.Request(suite.T(), "PATCH", envelope.Data.Links.Self, `{ "name": "Updated new envelope for testing" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedEnvelope controllers.EnvelopeResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedEnvelope)

	assert.Equal(suite.T(), envelope.Data.Note, updatedEnvelope.Data.Note)
	assert.Equal(suite.T(), "Updated new envelope for testing", updatedEnvelope.Data.Name)
}

func (suite *TestSuiteEnv) TestUpdateEnvelopeBroken() {
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{Name: "New envelope", Note: "Keks is a cuddly cat"})
	recorder := test.Request(suite.T(), "PATCH", envelope.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingEnvelope() {
	recorder := test.Request(suite.T(), "PATCH", "http://example.com/v1/envelopes/dcf472ba-a64e-4f0f-900e-a789319e432c", `{ "name": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteEnvelope() {
	e := createTestEnvelope(suite.T(), models.EnvelopeCreate{Name: "Delete me!"})
	r := test.Request(suite.T(), http.MethodDelete, e.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &r)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingEnvelope() {
	recorder := test.Request(suite.T(), "DELETE", "http://example.com/v1/envelopes/21a300da-d8b4-478d-8e85-95cb7982cbca", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteEnvelopeWithBody() {
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{Name: "Delete this envelope"})
	recorder := test.Request(suite.T(), "DELETE", envelope.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}
