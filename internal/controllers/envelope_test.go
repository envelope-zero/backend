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

func TestGetEnvelopes(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/envelopes", "")

	var response controllers.EnvelopeListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 1) {
		assert.FailNow(t, "Response does not have exactly 1 item")
	}

	assert.Equal(t, uint64(1), response.Data[0].CategoryID)
	assert.Equal(t, "Utilities", response.Data[0].Name)
	assert.Equal(t, "Energy & Water", response.Data[0].Note)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)
}

func TestNoEnvelopeNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/envelopes/2", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

// TestEnvelopeInvalidIDs verifies that on non-number requests for envelope IDs,
// the API returs a Bad Request status code.
func TestEnvelopeInvalidIDs(t *testing.T) {
	r := test.Request(t, "GET", "/v1/envelopes/-1985", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "GET", "/v1/envelopes/OhNoOurTable", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "PATCH", "/v1/envelopes/StupidLittleWalk", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "DELETE", "/v1/envelopes/25640ly", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelopeObject, savedEnvelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelopeObject)

	recorder = test.Request(t, "GET", envelopeObject.Data.Links.Self, "")
	test.DecodeResponse(t, &recorder, &savedEnvelope)

	assert.Equal(t, savedEnvelope, envelopeObject)
}

func TestCreateBrokenEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/envelopes", `{ "createdAt": "New Envelope", "note": "More tests for envelopes to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateEnvelopeNoCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/envelopes", `{ "categoryId": 5967 }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateEnvelopeNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/envelopes", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetEnvelope(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/envelopes/1", "")
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var envelopeObject, savedEnvelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelopeObject)

	recorder = test.Request(t, "GET", envelopeObject.Data.Links.Self, "")
	test.DecodeResponse(t, &recorder, &savedEnvelope)

	assert.Equal(t, savedEnvelope, envelopeObject)
}

// TestEnvelopeMonth verifies that the monthly calculations are correct.
func TestEnvelopeMonth(t *testing.T) {
	var envelopeMonth controllers.EnvelopeMonthResponse

	tests := []struct {
		path          string
		envelopeMonth models.EnvelopeMonth
	}{
		{
			"/v1/envelopes/1/2022-01",
			models.EnvelopeMonth{
				ID:         1,
				Name:       "Utilities",
				Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				Spent:      decimal.NewFromFloat(-10),
				Balance:    decimal.NewFromFloat(10.99),
				Allocation: decimal.NewFromFloat(20.99),
			},
		},
		{
			"/v1/envelopes/1/2022-02",
			models.EnvelopeMonth{
				ID:         1,
				Name:       "Utilities",
				Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
				Spent:      decimal.NewFromFloat(-5),
				Balance:    decimal.NewFromFloat(42.12),
				Allocation: decimal.NewFromFloat(47.12),
			},
		},
		{
			"/v1/envelopes/1/2022-03",
			models.EnvelopeMonth{
				ID:         1,
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
		assert.Equal(t, envelopeMonth.Data.ID, tt.envelopeMonth.ID)
		assert.Equal(t, envelopeMonth.Data.Name, tt.envelopeMonth.Name)
		assert.Equal(t, envelopeMonth.Data.Month, tt.envelopeMonth.Month)
		assert.True(t, envelopeMonth.Data.Spent.Equal(tt.envelopeMonth.Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Spent, envelopeMonth.Data.Spent, envelopeMonth.Data)
		assert.True(t, envelopeMonth.Data.Balance.Equal(tt.envelopeMonth.Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Balance, envelopeMonth.Data.Balance, envelopeMonth.Data)
		assert.True(t, envelopeMonth.Data.Allocation.Equal(tt.envelopeMonth.Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Allocation, envelopeMonth.Data.Allocation, envelopeMonth.Data)
	}
}

func TestEnvelopeMonthInvalid(t *testing.T) {
	// Test that non-parseable requests produce an error
	r := test.Request(t, "GET", "/v1/envelopes/1/Stonks!", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "GET", "/v1/envelopes/-17/2022-03", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

// TestEnvelopeMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func TestEnvelopeMonthZero(t *testing.T) {
	r := test.Request(t, "GET", "/v1/envelopes/1/0001-01", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestUpdateEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelope)

	path := fmt.Sprintf("/v1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new envelope for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedEnvelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &updatedEnvelope)

	assert.Equal(t, envelope.Data.Note, updatedEnvelope.Data.Note)
	assert.Equal(t, "Updated new envelope for testing", updatedEnvelope.Data.Name)
}

func TestUpdateEnvelopeBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelope)

	path := fmt.Sprintf("/v1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingEnvelope(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/envelopes/48902805", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteEnvelope(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/envelopes/1", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingEnvelope(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/envelopes/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteEnvelopeWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/envelopes", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope controllers.EnvelopeResponse
	test.DecodeResponse(t, &recorder, &envelope)

	path := fmt.Sprintf("/v1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
