package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestEnvelopeV3(t *testing.T, c models.EnvelopeCreate, expectedStatus ...int) controllers.EnvelopeResponseV3 {
	if c.CategoryID == uuid.Nil {
		c.CategoryID = suite.createTestCategory(models.CategoryCreate{}).Data.ID
	}

	if c.Name == "" {
		c.Name = uuid.NewString()
	}

	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	body := []models.EnvelopeCreate{c}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/envelopes", body)
	assertHTTPStatus(t, &r, expectedStatus...)

	var e controllers.EnvelopeCreateResponseV3
	suite.decodeResponse(&r, &e)

	if r.Code == http.StatusCreated {
		return e.Data[0]
	}

	return controllers.EnvelopeResponseV3{}
}

// TestEnvelopesV3DBClosed verifies that errors are processed correctly when
// the database is closed.
func (suite *TestSuiteStandard) TestEnvelopesV3DBClosed() {
	b := suite.createTestCategory(models.CategoryCreate{})

	tests := []struct {
		name string             // Name of the test
		test func(t *testing.T) // Code to run
	}{
		{
			"Creation fails",
			func(t *testing.T) {
				suite.createTestEnvelopeV3(t, models.EnvelopeCreate{CategoryID: b.Data.ID}, http.StatusInternalServerError)
			},
		},
		{
			"GET fails",
			func(t *testing.T) {
				recorder := test.Request(suite.controller, t, http.MethodGet, "http://example.com/v3/envelopes", "")
				assertHTTPStatus(t, &recorder, http.StatusInternalServerError)
				assert.Contains(t, test.DecodeError(t, recorder.Body.Bytes()), "there is a problem with the database connection")
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			suite.CloseDB()

			tt.test(t)
		})
	}
}

// TestEnvelopesV3Options verifies that OPTIONS requests are handled correctly.
func (suite *TestSuiteStandard) TestEnvelopesV3Options() {
	tests := []struct {
		name   string
		id     string // path at the Accounts endpoint to test
		status int    // Expected HTTP status code
	}{
		{"No Envelope with this ID", uuid.New().String(), http.StatusNotFound},
		{"Not a valid UUID", "NotParseableAsUUID", http.StatusBadRequest},
		{"Envelope exists", suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{}).Data.ID.String(), http.StatusNoContent},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", "http://example.com/v3/envelopes", tt.id)
			r := test.Request(suite.controller, t, http.MethodOptions, path, "")
			assertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestEnvelopesV3GetSingle verifies that requests for the resource endpoints are
// handled correctly.
func (suite *TestSuiteStandard) TestEnvelopesV3GetSingle() {
	e := suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{})

	tests := []struct {
		name   string
		id     string
		status int
		method string
	}{
		{"GET Existing Envelope", e.Data.ID.String(), http.StatusOK, http.MethodGet},
		{"GET ID nil", uuid.Nil.String(), http.StatusBadRequest, http.MethodGet},
		{"GET No Envelope with this ID", uuid.New().String(), http.StatusNotFound, http.MethodGet},
		{"GET Invalid ID (negative number)", "-56", http.StatusBadRequest, http.MethodGet},
		{"GET Invalid ID (positive number)", "23", http.StatusBadRequest, http.MethodGet},
		{"GET Invalid ID (string)", "notaUUID", http.StatusBadRequest, http.MethodGet},
		{"PATCH Invalid ID (negative number)", "-56", http.StatusBadRequest, http.MethodPatch},
		{"PATCH Invalid ID (positive number)", "23", http.StatusBadRequest, http.MethodPatch},
		{"PATCH Invalid ID (string)", "notaUUID", http.StatusBadRequest, http.MethodPatch},
		{"DELETE Invalid ID (negative number)", "-56", http.StatusBadRequest, http.MethodDelete},
		{"DELETE Invalid ID (positive number)", "23", http.StatusBadRequest, http.MethodDelete},
		{"DELETE Invalid ID (string)", "notaUUID", http.StatusBadRequest, http.MethodDelete},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, tt.method, fmt.Sprintf("http://example.com/v3/envelopes/%s", tt.id), "")

			var envelope controllers.EnvelopeResponseV3
			suite.decodeResponse(&r, &envelope)
			assertHTTPStatus(t, &r, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestEnvelopesV3GetFilter() {
	c1 := suite.createTestCategory(models.CategoryCreate{})
	c2 := suite.createTestCategory(models.CategoryCreate{})

	_ = suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
		Name:       "Groceries",
		Note:       "For the stuff bought in supermarkets",
		CategoryID: c1.Data.ID,
	})

	_ = suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
		Name:       "Hairdresser",
		Note:       "Because… Hair!",
		CategoryID: c2.Data.ID,
		Hidden:     true,
	})

	_ = suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
		Name:       "Stamps",
		Note:       "Because each stamp needs to go on an envelope. Hopefully it's not hairy",
		CategoryID: c2.Data.ID,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Category 2", fmt.Sprintf("category=%s", c2.Data.ID), 2},
		{"Category Not Existing", "category=e0f9ff7a-9f07-463c-bbd2-0d72d09d3cc6", 0},
		{"Empty Note", "note=", 0},
		{"Empty Name", "name=", 0},
		{"Name & Note", "name=Groceries&note=For the stuff bought in supermarkets", 1},
		{"Fuzzy name", "name=es", 2},
		{"Fuzzy note", "note=Because", 2},
		{"Not archived", "archived=false", 2},
		{"Archived", "archived=true", 1},
		{"Search for 'hair'", "search=hair", 2},
		{"Search for 'st'", "search=st", 2},
		{"Search for 'STUFF'", "search=STUFF", 1},
		{"Offset 2", "offset=2", 1},
		{"Offset 0, limit 2", "offset=0&limit=2", 2},
		{"Limit 4", "limit=4", 3},
		{"Limit 0", "limit=0", 0},
		{"Limit -1", "limit=-1", 3},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.EnvelopeListResponse
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v3/envelopes?%s", tt.query), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestEnvelopesV3CreateFails() {
	// Test envelope for uniqueness
	e := suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
		Name: "Unique Envelope Name for Category",
	})

	tests := []struct {
		name   string
		body   any
		status int // expected HTTP status
	}{
		{"Broken Body", `[{ "note": 2 }]`, http.StatusBadRequest},
		{"No body", "", http.StatusBadRequest},
		{
			"No Category",
			`[{ "note": "Some text" }]`,
			http.StatusBadRequest,
		},
		{
			"Non-existing Category",
			`[{ "categoryId": "ea85ad1a-3679-4ced-b83b-89566c12ece9" }]`,
			http.StatusBadRequest,
		},
		{
			"Duplicate name in Category",
			models.EnvelopeCreate{
				CategoryID: e.Data.CategoryID,
				Name:       e.Data.Name,
			},
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			recorder := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/envelopes", tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestEnvelopesV3Update() {
	envelope := suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{Name: "New envelope", Note: "Keks is a cuddly cat"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, envelope.Data.Links.Self, map[string]any{
		"name": "Updated new envelope for testing",
		"note": "",
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedEnvelope controllers.EnvelopeResponse
	suite.decodeResponse(&recorder, &updatedEnvelope)

	assert.Equal(suite.T(), "", updatedEnvelope.Data.Note)
	assert.Equal(suite.T(), "Updated new envelope for testing", updatedEnvelope.Data.Name)
}

func (suite *TestSuiteStandard) TestEnvelopesV3UpdateFails() {
	tests := []struct {
		name   string
		id     string
		body   any
		status int // expected response status
	}{
		{"Invalid type", "", `{"name": 2}`, http.StatusBadRequest},
		{"Broken JSON", "", `{ "name": 2" }`, http.StatusBadRequest},
		{"Non-existing account", uuid.New().String(), `{"name": 2}`, http.StatusNotFound},
		{"Set Category to uuid.Nil", "", models.EnvelopeCreate{}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				envelope := suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
					Name: "New Envelope",
					Note: "Auto-created for test",
				})

				tt.id = envelope.Data.ID.String()
			}

			// Update Account
			recorder = test.Request(suite.controller, t, http.MethodPatch, fmt.Sprintf("http://example.com/v3/envelopes/%s", tt.id), tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestEnvelopesV3Delete verifies all cases for Account deletions.
func (suite *TestSuiteStandard) TestEnvelopesV3Delete() {
	tests := []struct {
		name   string
		id     string
		status int // expected response status
	}{
		{"Success", "", http.StatusNoContent},
		{"Non-existing Envelope", uuid.New().String(), http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				// Create test Account
				e := suite.createTestEnvelopeV3(t, models.EnvelopeCreate{})
				tt.id = e.Data.ID.String()
			}

			// Delete Account
			recorder = test.Request(suite.controller, t, http.MethodDelete, fmt.Sprintf("http://example.com/v3/envelopes/%s", tt.id), "")
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestEnvelopesV3GetSorted verifies that Accounts are sorted by name.
func (suite *TestSuiteStandard) TestEnvelopesV3GetSorted() {
	e1 := suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
		Name: "Alphabetically first",
	})

	e2 := suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
		Name: "Second in creation, third in list",
	})

	e3 := suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
		Name: "First is alphabetically second",
	})

	e4 := suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{
		Name: "Zulu is the last one",
	})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v3/envelopes", "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)

	var envelopes controllers.EnvelopeListResponseV3
	suite.decodeResponse(&r, &envelopes)

	if !assert.Len(suite.T(), envelopes.Data, 4) {
		assert.FailNow(suite.T(), "Envelope list has wrong length")
	}

	assert.Equal(suite.T(), e1.Data.Name, envelopes.Data[0].Name)
	assert.Equal(suite.T(), e2.Data.Name, envelopes.Data[2].Name)
	assert.Equal(suite.T(), e3.Data.Name, envelopes.Data[1].Name)
	assert.Equal(suite.T(), e4.Data.Name, envelopes.Data[3].Name)
}

func (suite *TestSuiteStandard) TestEnvelopesV3Pagination() {
	for i := 0; i < 10; i++ {
		suite.createTestEnvelopeV3(suite.T(), models.EnvelopeCreate{Name: fmt.Sprint(i)})
	}

	tests := []struct {
		name          string
		offset        uint
		limit         int
		expectedCount int
		expectedTotal int64
	}{
		{"All", 0, -1, 10, 10},
		{"First 5", 0, 5, 5, 10},
		{"Last 5", 5, -1, 5, 10},
		{"Offset 3", 3, -1, 7, 10},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v3/envelopes?offset=%d&limit=%d", tt.offset, tt.limit), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)

			var envelopes controllers.EnvelopeListResponseV3
			suite.decodeResponse(&r, &envelopes)

			assert.Equal(suite.T(), tt.offset, envelopes.Pagination.Offset)
			assert.Equal(suite.T(), tt.limit, envelopes.Pagination.Limit)
			assert.Equal(suite.T(), tt.expectedCount, envelopes.Pagination.Count)
			assert.Equal(suite.T(), tt.expectedTotal, envelopes.Pagination.Total)
		})
	}
}
