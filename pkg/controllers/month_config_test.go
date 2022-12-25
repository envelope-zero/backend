package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/types"
	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestMonthConfig(envelopeID uuid.UUID, month types.Month, c models.MonthConfigCreate, expectedStatus ...int) controllers.MonthConfigResponse {
	if envelopeID == uuid.Nil {
		envelopeID = suite.createTestEnvelope(models.EnvelopeCreate{Name: "Transaction Test Envelope"}).Data.ID
	}

	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	path := fmt.Sprintf("http://example.com/v1/month-configs/%s/%s", envelopeID, month.String())
	r := test.Request(suite.controller, suite.T(), http.MethodPost, path, c)
	suite.assertHTTPStatus(&r, expectedStatus...)

	var mc controllers.MonthConfigResponse
	suite.decodeResponse(&r, &mc)

	return mc
}

func (suite *TestSuiteStandard) TestMonthConfigsEmptyList() {
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/month-configs", "")

	var l controllers.MonthConfigListResponse
	suite.decodeResponse(&r, &l)

	// Verify that the list is an empty list, not null
	suite.Assert().NotNil(l.Data)
	suite.Assert().Empty(l.Data)
}

func (suite *TestSuiteStandard) TestMonthConfigsCreate() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})
	someMonth := types.NewMonth(2020, 3)

	tests := []struct {
		name       string
		envelopeID uuid.UUID
		month      types.Month
		status     int
	}{
		{"Standard create", envelope.Data.ID, someMonth, http.StatusCreated},
		{"duplicate config for same envelope and month", envelope.Data.ID, someMonth, http.StatusBadRequest},
		{"No envelope", uuid.New(), someMonth, http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			_ = suite.createTestMonthConfig(tt.envelopeID, tt.month, models.MonthConfigCreate{}, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsCreateInvalid() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})

	tests := []struct {
		name       string
		envelopeID string
		month      string
		body       string
	}{
		{"Invalid Body", envelope.Data.ID.String(), "2022-03", `{"name": "not valid body"`},
		{"Invaid UUID", "not a uuid", "2017-04", ""},
		{"Invalid month", envelope.Data.ID.String(), "September Seventy Seven", ""},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v1/month-configs", tt.envelopeID, tt.month)

			recorder := test.Request(suite.controller, suite.T(), http.MethodPost, path, tt.body)
			assert.Equal(t, http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsGet() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})
	someMonth := types.NewMonth(2020, 3)

	_ = suite.createTestMonthConfig(envelope.Data.ID, someMonth, models.MonthConfigCreate{})

	tests := []struct {
		name       string
		envelopeID string
		month      string
		status     int
	}{
		{"Standard get", envelope.Data.ID.String(), someMonth.String(), http.StatusOK},
		{"No envelope", uuid.New().String(), someMonth.String(), http.StatusNotFound},
		{"Invalid UUID", "Not a UUID", someMonth.String(), http.StatusBadRequest},
		{"Invalid month", envelope.Data.ID.String(), "2193-1", http.StatusBadRequest},
		{"No MonthConfig", envelope.Data.ID.String(), "0333-11", http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v1/month-configs", tt.envelopeID, tt.month)

			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, path, "")
			assert.Equal(t, tt.status, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

			if tt.status == http.StatusOK {
				var mConfig controllers.MonthConfigResponse
				suite.decodeResponse(&recorder, &mConfig)

				selfLink := fmt.Sprintf("http://example.com/v1/month-configs/%s/%s", tt.envelopeID, tt.month)
				assert.Equal(t, selfLink, mConfig.Data.Links.Self, "Request ID %s", recorder.Header().Get("x-request-id"))

				envelopeLink := fmt.Sprintf("http://example.com/v1/envelopes/%s", tt.envelopeID)
				assert.Equal(t, envelopeLink, mConfig.Data.Links.Envelope, "Request ID %s", recorder.Header().Get("x-request-id"))
			}
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsCreateDBError() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})
	suite.CloseDB()

	_ = suite.createTestMonthConfig(envelope.Data.ID, types.NewMonth(2020, 3), models.MonthConfigCreate{}, http.StatusInternalServerError)
}

func (suite *TestSuiteStandard) TestMonthConfigsOptions() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})
	_ = suite.createTestMonthConfig(
		envelope.Data.ID,
		types.NewMonth(2014, 5),
		models.MonthConfigCreate{},
	)

	tests := []struct {
		name     string
		envelope string
		month    string
		status   int
		errMsg   string
	}{
		{"Bad Envelope ID", "Definitely-Not-A-UUID", "1984-03", http.StatusBadRequest, "not a valid UUID"},
		{"Invalid Month", envelope.Data.ID.String(), "2000-00", http.StatusBadRequest, "Could not parse the specified month"},
		{"No envelope", uuid.New().String(), "1984-03", http.StatusNoContent, ""},
		{"No MonthConfig", envelope.Data.ID.String(), "1984-03", http.StatusNoContent, ""},
		{"Existing", envelope.Data.ID.String(), "2014-05", http.StatusNoContent, ""},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v1/month-configs", tt.envelope, tt.month)
			recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
			assert.Equal(t, tt.status, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

			if tt.status != http.StatusNoContent {
				assert.Contains(t, test.DecodeError(suite.T(), recorder.Body.Bytes()), tt.errMsg)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsGetList() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})
	_ = suite.createTestMonthConfig(
		envelope.Data.ID,
		types.NewMonth(2007, 10),
		models.MonthConfigCreate{},
	)

	_ = suite.createTestMonthConfig(
		envelope.Data.ID,
		types.NewMonth(3017, 10),
		models.MonthConfigCreate{},
	)

	tests := []struct {
		name   string
		query  string
		status int
		length int
	}{
		{"No envelope", fmt.Sprintf("envelope=%s&month=%s", uuid.New().String(), "1984-03"), http.StatusOK, 0},
		{"No MonthConfig", fmt.Sprintf("envelope=%s&month=%s", envelope.Data.ID.String(), "1984-03"), http.StatusOK, 0},
		{"Exact MonthConfig", fmt.Sprintf("envelope=%s&month=%s", envelope.Data.ID.String(), "2007-10"), http.StatusOK, 1},
		{"Month only", "month=2007-10", http.StatusOK, 1},
		{"Envelope ID only", fmt.Sprintf("envelope=%s", envelope.Data.ID.String()), http.StatusOK, 2},
		{"Bad Envelope ID", fmt.Sprintf("envelope=%s&month=%s", "Definitely-Not-A-UUID", "1984-03"), http.StatusBadRequest, 0},
		{"Invalid Month", fmt.Sprintf("envelope=%s&month=%s", envelope.Data.ID.String(), "2000-00"), http.StatusBadRequest, 0},
		{"Invalid query string", "envelope=;", http.StatusBadRequest, 0},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s?%s", "http://example.com/v1/month-configs", tt.query)
			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, path, "")
			assert.Equal(t, tt.status, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

			var l controllers.MonthConfigListResponse
			suite.decodeResponse(&recorder, &l)
			assert.Len(t, l.Data, tt.length)
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsGetDBError() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/month-configs", "")
	suite.Assert().Equal(http.StatusInternalServerError, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestMonthConfigsDelete() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})
	someMonth := types.NewMonth(2020, 3)

	_ = suite.createTestMonthConfig(envelope.Data.ID, someMonth, models.MonthConfigCreate{})

	tests := []struct {
		name       string
		envelopeID string
		month      string
		status     int
	}{
		{"Standard get", envelope.Data.ID.String(), someMonth.String(), http.StatusNoContent},
		{"No envelope", uuid.New().String(), someMonth.String(), http.StatusNotFound},
		{"Invalid UUID", "Not a UUID", someMonth.String(), http.StatusBadRequest},
		{"Invalid month", envelope.Data.ID.String(), "2193-1", http.StatusBadRequest},
		{"No MonthConfig", envelope.Data.ID.String(), "0333-11", http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v1/month-configs", tt.envelopeID, tt.month)

			recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, path, "")
			assert.Equal(t, tt.status, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestUpdateMonthConfig() {
	mConfig := suite.createTestMonthConfig(uuid.Nil, types.NewMonth(time.Now().Year(), time.Now().Month()), models.MonthConfigCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, mConfig.Data.Links.Self, models.MonthConfigCreate{
		OverspendMode: "AFFECT_ENVELOPE",
	})
	suite.assertHTTPStatus(&recorder, http.StatusOK)

	var updatedMonthConfig controllers.MonthConfigResponse
	suite.decodeResponse(&recorder, &updatedMonthConfig)

	var mode models.OverspendMode = "AFFECT_ENVELOPE"
	assert.Equal(suite.T(), mode, updatedMonthConfig.Data.OverspendMode)
}

func (suite *TestSuiteStandard) TestMonthConfigsUpdateInvalid() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})
	mConfig := suite.createTestMonthConfig(envelope.Data.ID, types.NewMonth(2022, 3), models.MonthConfigCreate{})

	tests := []struct {
		name       string
		envelopeID string
		month      string
		body       string
		status     int
	}{
		{"Invalid Body", envelope.Data.ID.String(), mConfig.Data.Month.String(), `{"name": "not valid body"`, http.StatusBadRequest},
		{"Invaid UUID", "not a uuid", "2017-04", "", http.StatusBadRequest},
		{"Invalid month", envelope.Data.ID.String(), "September Seventy Seven", "", http.StatusBadRequest},
		{"No envelope", uuid.NewString(), mConfig.Data.Month.String(), "", http.StatusNotFound},
		{"No month config", envelope.Data.ID.String(), "0137-12", "", http.StatusNotFound},
		{"Broken values", envelope.Data.ID.String(), mConfig.Data.Month.String(), `{"overspendMode": 2 }`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v1/month-configs", tt.envelopeID, tt.month)

			recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, path, tt.body)
			assert.Equal(t, tt.status, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestUpdateMonthConfigBrokenJSON() {
	mConfig := suite.createTestMonthConfig(uuid.Nil, types.NewMonth(time.Now().Year(), time.Now().Month()), models.MonthConfigCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, mConfig.Data.Links.Self, `{ test`)
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
}
