package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestMonthConfigsV3GetSingle() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})
	someMonth := types.NewMonth(2020, 3)

	suite.controller.DB.Create(&models.MonthConfig{
		MonthConfigCreate: models.MonthConfigCreate{
			Note: "This is to test GET with existing Month Config",
		},
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
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v3/envelopes", tt.envelopeID, tt.month)

			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, path, "")
			assertHTTPStatus(t, &recorder, tt.status)

			if tt.status == http.StatusOK {
				var mConfig controllers.MonthConfigResponseV3
				suite.decodeResponse(&recorder, &mConfig)

				selfLink := fmt.Sprintf("http://example.com/v3/envelopes/%s/%s", tt.envelopeID, tt.month)
				assert.Equal(t, selfLink, mConfig.Data.Links.Self, "Request ID %s", recorder.Header().Get("x-request-id"))

				envelopeLink := fmt.Sprintf("http://example.com/v3/envelopes/%s", tt.envelopeID)
				assert.Equal(t, envelopeLink, mConfig.Data.Links.Envelope, "Request ID %s", recorder.Header().Get("x-request-id"))
			}
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsV3Options() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})

	tests := []struct {
		name     string
		envelope string
		month    string
		status   int
		errMsg   string
	}{
		{"Bad Envelope ID", "Definitely-Not-A-UUID", "1984-03", http.StatusBadRequest, "not a valid UUID"},
		{"Invalid Month", envelope.Data.ID.String(), "2000-00", http.StatusBadRequest, "could not parse: parsing time \"2000-00\": month out of range"},
		{"No envelope", uuid.New().String(), "1984-03", http.StatusNoContent, ""},
		{"No MonthConfig", envelope.Data.ID.String(), "1984-03", http.StatusNoContent, ""},
		{"Existing", envelope.Data.ID.String(), "2014-05", http.StatusNoContent, ""},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v3/envelopes", tt.envelope, tt.month)
			recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
			assertHTTPStatus(t, &recorder, tt.status)

			if tt.status != http.StatusNoContent {
				assert.Contains(t, test.DecodeError(suite.T(), recorder.Body.Bytes()), tt.errMsg)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsV3Update() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})
	month := types.NewMonth(time.Now().Year(), time.Now().Month())

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, fmt.Sprintf("http://example.com/v3/envelopes/%s/%s", envelope.Data.ID, month), models.MonthConfigCreate{
		Note: "This is the updated note",
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedMonthConfig controllers.MonthConfigResponseV3
	suite.decodeResponse(&recorder, &updatedMonthConfig)
	assert.Equal(suite.T(), "This is the updated note", updatedMonthConfig.Data.Note)
}

func (suite *TestSuiteStandard) TestMonthConfigsV3UpdateFails() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})
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
			path := fmt.Sprintf("%s/%s/%s", "http://example.com/v3/envelopes", tt.envelopeID, tt.month)

			recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, path, tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestMonthConfigsV3UpdateAllocationCreatesResource() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})
	month := types.NewMonth(time.Now().Year(), time.Now().Month())

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, fmt.Sprintf("http://example.com/v3/envelopes/%s/%s", envelope.Data.ID, month), controllers.MonthConfigCreateV3{
		Note:       "This is the updated note",
		Allocation: decimal.NewFromFloat(17.32),
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedMonthConfig controllers.MonthConfigResponseV3
	suite.decodeResponse(&recorder, &updatedMonthConfig)
	assert.Equal(suite.T(), "This is the updated note", updatedMonthConfig.Data.Note)
	assert.Equal(suite.T(), decimal.NewFromFloat(17.32), updatedMonthConfig.Data.Allocation)

	// Verify that the allocation is set to the correct value
	a := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			Month:      month,
			EnvelopeID: envelope.Data.ID,
		},
	}

	err := suite.controller.DB.First(&a).Error
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), decimal.NewFromFloat(17.32), a.Amount)
}
