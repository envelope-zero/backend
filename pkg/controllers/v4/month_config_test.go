package v4_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v5/internal/types"
	v4 "github.com/envelope-zero/backend/v5/pkg/controllers/v4"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func patchTestMonthConfig(t *testing.T, envelopeID uuid.UUID, month types.Month, c v4.MonthConfigEditable, expectedStatus ...int) v4.MonthConfigResponse {
	if envelopeID == uuid.Nil {
		envelopeID = createTestEnvelope(t, v4.EnvelopeEditable{Name: "Transaction Test Envelope"}).Data.ID
	}

	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusOK)
	}

	path := fmt.Sprintf("http://example.com/v4/envelopes/%s/%s", envelopeID, month.String())
	r := test.Request(t, http.MethodPatch, path, c)
	test.AssertHTTPStatus(t, &r, expectedStatus...)

	var mc v4.MonthConfigResponse
	test.DecodeResponse(t, &r, &mc)

	return mc
}

func (suite *TestSuiteStandard) TestMonthConfigsGetSingle() {
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{})
	someMonth := types.NewMonth(2020, 3)

	models.DB.Create(&models.MonthConfig{
		Note:       "This is to test GET with existing Month Config",
		EnvelopeID: envelope.Data.ID,
		Month:      someMonth,
	})

	tests := []struct {
		name       string
		envelopeID string
		month      string
		status     int
	}{
		{"Month Config exists", envelope.Data.ID.String(), someMonth.String(), http.StatusOK},
		{"No Month Config exists", envelope.Data.ID.String(), "0333-11", http.StatusOK},
		{"No envelope", uuid.New().String(), someMonth.String(), http.StatusNotFound},
		{"Invalid UUID", "Not a UUID", someMonth.String(), http.StatusBadRequest},
		{"Invalid month", envelope.Data.ID.String(), "2193-1", http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v4/envelopes", tt.envelopeID, tt.month)

			recorder := test.Request(suite.T(), http.MethodGet, path, "")
			test.AssertHTTPStatus(t, &recorder, tt.status)

			if tt.status == http.StatusOK {
				var mConfig v4.MonthConfigResponse
				test.DecodeResponse(t, &recorder, &mConfig)

				selfLink := fmt.Sprintf("http://example.com/v4/envelopes/%s/%s", tt.envelopeID, tt.month)
				assert.Equal(t, selfLink, mConfig.Data.Links.Self, "Request ID %s", recorder.Header().Get("x-request-id"))

				envelopeLink := fmt.Sprintf("http://example.com/v4/envelopes/%s", tt.envelopeID)
				assert.Equal(t, envelopeLink, mConfig.Data.Links.Envelope, "Request ID %s", recorder.Header().Get("x-request-id"))
			}
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsOptions() {
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{})

	tests := []struct {
		name     string
		envelope string
		month    string
		status   int
		errMsg   string
	}{
		{"Bad Envelope ID", "Definitely-Not-A-UUID", "1984-03", http.StatusBadRequest, "invalid UUID length: 21"},
		{"Invalid Month", envelope.Data.ID.String(), "2000-00", http.StatusBadRequest, "parsing time \"2000-00\": month out of range"},
		{"No envelope", uuid.New().String(), "1984-03", http.StatusNoContent, ""},
		{"No MonthConfig", envelope.Data.ID.String(), "1984-03", http.StatusNoContent, ""},
		{"Existing", envelope.Data.ID.String(), "2014-05", http.StatusNoContent, ""},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v4/envelopes", tt.envelope, tt.month)
			recorder := test.Request(suite.T(), http.MethodOptions, path, "")
			test.AssertHTTPStatus(t, &recorder, tt.status)

			if tt.status != http.StatusNoContent {
				var response struct {
					Error string `json:"error"`
				}
				test.DecodeResponse(t, &recorder, &response)
				assert.Contains(t, response.Error, tt.errMsg)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsUpdate() {
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{})
	month := types.NewMonth(time.Now().Year(), time.Now().Month())

	recorder := test.Request(suite.T(), http.MethodPatch, fmt.Sprintf("http://example.com/v4/envelopes/%s/%s", envelope.Data.ID, month), v4.MonthConfigEditable{
		Note: "This is the updated note",
	})
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedMonthConfig v4.MonthConfigResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedMonthConfig)
	assert.Equal(suite.T(), "This is the updated note", updatedMonthConfig.Data.Note)
}

func (suite *TestSuiteStandard) TestMonthConfigsUpdateFails() {
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{})
	month := types.NewMonth(2022, 3)

	tests := []struct {
		name       string
		envelopeID string
		month      string
		body       string
		status     int
	}{
		{"Invalid Body", envelope.Data.ID.String(), month.String(), `{"name": "not valid body"`, http.StatusBadRequest},
		{"Invaid UUID", "not a uuid", "2017-04", "", http.StatusBadRequest},
		{"Invalid month", envelope.Data.ID.String(), "September Seventy Seven", "", http.StatusBadRequest},
		{"No envelope", uuid.NewString(), month.String(), "", http.StatusNotFound},
		{"No month config", envelope.Data.ID.String(), "1137-12", `{"note": "This implicitly creates a Month Config"}`, http.StatusOK},
		{"Broken values", envelope.Data.ID.String(), month.String(), `{"note": 2 }`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v4/envelopes", tt.envelopeID, tt.month)

			recorder := test.Request(suite.T(), http.MethodPatch, path, tt.body)
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}
