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

type EnvelopeListResponse struct {
	test.APIResponse
	Data []models.Envelope
}

type EnvelopeDetailResponse struct {
	test.APIResponse
	Data models.Envelope
}

func TestGetEnvelopes(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes", "")

	var response EnvelopeListResponse
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 1) {
		assert.FailNow(t, "Response does not have exactly 1 item")
	}

	assert.Equal(t, uint64(1), response.Data[0].CategoryID)
	assert.Equal(t, "Utilities", response.Data[0].Name)
	assert.Equal(t, "Energy & Water", response.Data[0].Note)

	diff := time.Now().Sub(response.Data[0].CreatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)

	diff = time.Now().Sub(response.Data[0].UpdatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)
}

func TestNoEnvelopeNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/2", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiEnvelope EnvelopeDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiEnvelope)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbEnvelope models.Envelope
	models.DB.First(&dbEnvelope, apiEnvelope.Data.ID)

	assert.Equal(t, dbEnvelope, apiEnvelope.Data)
}

func TestCreateBrokenEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", `{ "createdAt": "New Envelope", "note": "More tests for envelopes to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateEnvelopeNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetEnvelope(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/1", "")
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var envelope EnvelopeDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&envelope)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbEnvelope models.Envelope
	models.DB.First(&dbEnvelope, envelope.Data.ID)

	assert.Equal(t, dbEnvelope, envelope.Data)
}

func TestUpdateEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope EnvelopeDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&envelope)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new envelope for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedEnvelope EnvelopeDetailResponse
	err = json.NewDecoder(recorder.Body).Decode(&updatedEnvelope)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, envelope.Data.Note, updatedEnvelope.Data.Note)
	assert.Equal(t, "Updated new envelope for testing", updatedEnvelope.Data.Name)
}

func TestUpdateEnvelopeBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope EnvelopeDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&envelope)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingEnvelope(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/1/categories/1/envelopes/48902805", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteEnvelope(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/categories/1/envelopes/1", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingEnvelope(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/categories/1/envelopes/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteEnvelopeWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope EnvelopeDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&envelope)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
