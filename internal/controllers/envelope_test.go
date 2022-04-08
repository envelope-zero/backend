package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/shopspring/decimal"
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

type EnvelopeMonthResponse struct {
	test.APIResponse
	Data struct {
		Month time.Time       `json:"month"`
		Spent decimal.Decimal `json:"spent"`
	}
}

func TestGetEnvelopes(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes", "")

	var response EnvelopeListResponse
	test.DecodeResponse(t, &recorder, &response)

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

// TestEnvelopeInvalidIDs verifies that on non-number requests for envelope IDs,
// the API returs a Bad Request status code.
func TestEnvelopeInvalidIDs(t *testing.T) {
	r := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/-1985", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/OhNoOurTable", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

// TestNonexistingCategoryEnvelopes404 is a regression test for https://github.com/envelope-zero/backend/issues/89.
//
// It verifies that for a non-existing category, the envelopes endpoint raises a 404
// instead of returning an empty list.
func TestNonexistingCategoryEnvelopes404(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/999/envelopes", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

// TestNonexistingBudgetEnvelopes404 is a regression test for https://github.com/envelope-zero/backend/issues/89.
//
// It verifies that for a non-existing budget, no matter if the category with the ID exists,
// the envelopes endpoint raises a 404 instead of returning an empty list.
func TestNonexistingBudgetEnvelopes404(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/999/categories/1/envelopes", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

// TestEnvelopeParentChecked is a regression test for https://github.com/envelope-zero/backend/issues/90.
//
// It verifies that the envelope details endpoint for a category only returns envelopes that belong to the
// category.
func TestEnvelopeParentChecked(t *testing.T) {
	r := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "Testing category" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var category CategoryDetailResponse
	test.DecodeResponse(t, &r, &category)

	path := fmt.Sprintf("/v1/budgets/1/categories/%v", category.Data.ID)
	r = test.Request(t, "GET", path+"/envelopes/1", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &r)

	r = test.Request(t, "DELETE", path, "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &r)
}

func TestCreateEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiEnvelope EnvelopeDetailResponse
	test.DecodeResponse(t, &recorder, &apiEnvelope)

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
	test.DecodeResponse(t, &recorder, &envelope)

	var dbEnvelope models.Envelope
	models.DB.First(&dbEnvelope, envelope.Data.ID)

	assert.Equal(t, dbEnvelope, envelope.Data)
}

func TestEnvelopeMonth(t *testing.T) {
	var envelopeMonth EnvelopeMonthResponse

	r := test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/1?month=2022-01", "")
	test.AssertHTTPStatus(t, http.StatusOK, &r)
	spent := decimal.NewFromFloat(-10)
	test.DecodeResponse(t, &r, &envelopeMonth)
	assert.True(t, envelopeMonth.Data.Spent.Equal(spent), "Month calculation for 2022-01 is wrong: should be %v, but is %v", spent, envelopeMonth.Data.Spent)

	r = test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/1?month=2022-02", "")
	test.AssertHTTPStatus(t, http.StatusOK, &r)
	spent = decimal.NewFromFloat(-5)
	test.DecodeResponse(t, &r, &envelopeMonth)
	assert.True(t, envelopeMonth.Data.Spent.Equal(spent), "Month calculation for 2022-02 is wrong: should be %v, but is %v", spent, envelopeMonth.Data.Spent)

	r = test.Request(t, "GET", "/v1/budgets/1/categories/1/envelopes/1?month=2022-03", "")
	test.AssertHTTPStatus(t, http.StatusOK, &r)
	spent = decimal.NewFromFloat(-15)
	test.DecodeResponse(t, &r, &envelopeMonth)
	assert.True(t, envelopeMonth.Data.Spent.Equal(spent), "Month calculation for 2022-03 is wrong: should be %v, but is %v", spent, envelopeMonth.Data.Spent)
}

func TestUpdateEnvelope(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope EnvelopeDetailResponse
	test.DecodeResponse(t, &recorder, &envelope)

	path := fmt.Sprintf("/v1/budgets/1/categories/1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new envelope for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedEnvelope EnvelopeDetailResponse
	test.DecodeResponse(t, &recorder, &updatedEnvelope)

	assert.Equal(t, envelope.Data.Note, updatedEnvelope.Data.Note)
	assert.Equal(t, "Updated new envelope for testing", updatedEnvelope.Data.Name)
}

func TestUpdateEnvelopeBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories/1/envelopes", `{ "name": "New Envelope", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var envelope EnvelopeDetailResponse
	test.DecodeResponse(t, &recorder, &envelope)

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
	test.DecodeResponse(t, &recorder, &envelope)

	path := fmt.Sprintf("/v1/budgets/1/categories/1/envelopes/%v", envelope.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
