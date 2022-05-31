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

func createTestEnvelope(t *testing.T, c models.EnvelopeCreate) controllers.EnvelopeResponse {
	r := test.Request(t, "POST", "http://example.com/v1/envelopes", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var e controllers.EnvelopeResponse
	test.DecodeResponse(t, &r, &e)

	return e
}

func TestGetEnvelopes(t *testing.T) {
	recorder := test.Request(t, "GET", "http://example.com/v1/envelopes", "")

	var response controllers.EnvelopeListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 1) {
		assert.FailNow(t, "Response does not have exactly 1 item")
	}

	assert.Equal(t, "Utilities", response.Data[0].Name)
	assert.Equal(t, "Energy & Water", response.Data[0].Note)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)
}

func TestNoEnvelopeNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "http://example.com/v1/envelopes/828f2483-dabd-4267-a223-e34b5f171978", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestEnvelopeInvalidIDs(t *testing.T) {
	/*
	 *  GET
	 */
	r := test.Request(t, http.MethodGet, "http://example.com/v1/envelopes/-56", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "http://example.com/v1/envelopes/notANumber", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "http://example.com/v1/envelopes/23", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "http://example.com/v1/envelopes/d19a622f-broken-uuid/2017-09", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(t, http.MethodPatch, "http://example.com/v1/envelopes/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodPatch, "http://example.com/v1/envelopes/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(t, http.MethodDelete, "http://example.com/v1/envelopes/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodDelete, "http://example.com/v1/envelopes/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "http://example.com/v1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelopeObject, savedEnvelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelopeObject)

	recorder = test.Request(t, "GET", envelopeObject.Data.Links.Self, "")
	test.DecodeResponse(t, &recorder, &savedEnvelope)

	assert.Equal(t, savedEnvelope, envelopeObject)
}

func TestCreateBrokenEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "http://example.com/v1/envelopes", `{ "createdAt": "New Envelope", "note": "More tests for envelopes to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateEnvelopeNoCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "http://example.com/v1/envelopes", `{ "categoryId": "5f0cd7b9-9788-4871-96f8-c816c9ae338a" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateEnvelopeNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "http://example.com/v1/envelopes", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetEnvelope(t *testing.T) {
	_ = createTestEnvelope(t, models.EnvelopeCreate{Name: "This only tests creation"})
}

// TestEnvelopeMonth verifies that the monthly calculations are correct.
func TestEnvelopeMonth(t *testing.T) {
	var envelopeList controllers.EnvelopeListResponse
	r := test.Request(t, http.MethodGet, "http://example.com/v1/envelopes", "")
	test.DecodeResponse(t, &r, &envelopeList)

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
		r := test.Request(t, "GET", tt.path, "")
		test.AssertHTTPStatus(t, http.StatusOK, &r)

		test.DecodeResponse(t, &r, &envelopeMonth)
		assert.Equal(t, envelopeMonth.Data.Name, tt.envelopeMonth.Name)
		assert.Equal(t, envelopeMonth.Data.Month, tt.envelopeMonth.Month)
		assert.True(t, envelopeMonth.Data.Spent.Equal(tt.envelopeMonth.Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Spent, envelopeMonth.Data.Spent, envelopeMonth.Data)
		assert.True(t, envelopeMonth.Data.Balance.Equal(tt.envelopeMonth.Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Balance, envelopeMonth.Data.Balance, envelopeMonth.Data)
		assert.True(t, envelopeMonth.Data.Allocation.Equal(tt.envelopeMonth.Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Allocation, envelopeMonth.Data.Allocation, envelopeMonth.Data)
	}
}

func TestEnvelopeMonthInvalid(t *testing.T) {
	var envelopeList controllers.EnvelopeListResponse
	r := test.Request(t, http.MethodGet, "http://example.com/v1/envelopes", "")
	test.DecodeResponse(t, &r, &envelopeList)

	// Test that non-parseable requests produce an error
	r = test.Request(t, "GET", fmt.Sprintf("http://example.com/v1/envelopes/%s/Stonks!", envelopeList.Data[0].ID), "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

// TestEnvelopeMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func TestEnvelopeMonthZero(t *testing.T) {
	e := createTestEnvelope(t, models.EnvelopeCreate{})
	r := test.Request(t, http.MethodGet, fmt.Sprintf("%s/0001-01", e.Data.Links.Self), "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestUpdateEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "http://example.com/v1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelope)

	path := fmt.Sprintf("http://example.com/v1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new envelope for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedEnvelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &updatedEnvelope)

	assert.Equal(t, envelope.Data.Note, updatedEnvelope.Data.Note)
	assert.Equal(t, "Updated new envelope for testing", updatedEnvelope.Data.Name)
}

func TestUpdateEnvelopeBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "http://example.com/v1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelope)

	path := fmt.Sprintf("http://example.com/v1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingEnvelope(t *testing.T) {
	recorder := test.Request(t, "PATCH", "http://example.com/v1/envelopes/dcf472ba-a64e-4f0f-900e-a789319e432c", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteEnvelope(t *testing.T) {
	e := createTestEnvelope(t, models.EnvelopeCreate{Name: "Delete me!"})
	r := test.Request(t, http.MethodDelete, e.Data.Links.Self, "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &r)
}

func TestDeleteNonExistingEnvelope(t *testing.T) {
	recorder := test.Request(t, "DELETE", "http://example.com/v1/envelopes/21a300da-d8b4-478d-8e85-95cb7982cbca", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteEnvelopeWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "http://example.com/v1/envelopes", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelope)

	path := fmt.Sprintf("http://example.com/v1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
