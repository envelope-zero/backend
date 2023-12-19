package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestEnvelopeV3(t *testing.T, c controllers.EnvelopeCreateV3, expectedStatus ...int) controllers.EnvelopeResponseV3 {
	if c.CategoryID == uuid.Nil {
		c.CategoryID = suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{}).Data.ID
	}

	if c.Name == "" {
		c.Name = uuid.NewString()
	}

	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	body := []controllers.EnvelopeCreateV3{c}

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
	b := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{})

	tests := []struct {
		name string             // Name of the test
		test func(t *testing.T) // Code to run
	}{
		{
			"Creation fails",
			func(t *testing.T) {
				suite.createTestEnvelopeV3(t, controllers.EnvelopeCreateV3{CategoryID: b.Data.ID}, http.StatusInternalServerError)
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
		{"Envelope exists", suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{}).Data.ID.String(), http.StatusNoContent},
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
	e := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})

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
	c1 := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{})
	c2 := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{})

	_ = suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
		Name:       "Groceries",
		Note:       "For the stuff bought in supermarkets",
		CategoryID: c1.Data.ID,
	})

	_ = suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
		Name:       "Hairdresser",
		Note:       "Because… Hair!",
		CategoryID: c2.Data.ID,
		Archived:   true,
	})

	_ = suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
		Name:       "Stamps",
		Note:       "Because each stamp needs to go on an envelope. Hopefully it's not hairy",
		CategoryID: c2.Data.ID,
	})

	tests := []struct {
		name      string
		query     string
		len       int
		checkFunc func(t *testing.T, envelopes []controllers.EnvelopeV3)
	}{
		{"Category 2", fmt.Sprintf("category=%s", c2.Data.ID), 2, nil},
		{"Category Not Existing", "category=e0f9ff7a-9f07-463c-bbd2-0d72d09d3cc6", 0, nil},
		{"Empty Note", "note=", 0, nil},
		{"Empty Name", "name=", 0, nil},
		{"Name & Note", "name=Groceries&note=For the stuff bought in supermarkets", 1, nil},
		{"Fuzzy name", "name=es", 2, nil},
		{"Fuzzy note", "note=Because", 2, nil},
		{"Not archived", "archived=false", 2, func(t *testing.T, envelopes []controllers.EnvelopeV3) {
			for _, e := range envelopes {
				assert.False(t, e.Archived)
			}
		}},
		{"Archived", "archived=true", 1, func(t *testing.T, envelopes []controllers.EnvelopeV3) {
			for _, e := range envelopes {
				assert.True(t, e.Archived)
			}
		}},
		{"Search for 'hair'", "search=hair", 2, nil},
		{"Search for 'st'", "search=st", 2, nil},
		{"Search for 'STUFF'", "search=STUFF", 1, nil},
		{"Offset 2", "offset=2", 1, nil},
		{"Offset 0, limit 2", "offset=0&limit=2", 2, nil},
		{"Limit 4", "limit=4", 3, nil},
		{"Limit 0", "limit=0", 0, nil},
		{"Limit -1", "limit=-1", 3, nil},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.EnvelopeListResponseV3
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v3/envelopes?%s", tt.query), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestEnvelopesV3CreateFails() {
	// Test envelope for uniqueness
	e := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
		Name: "Unique Envelope Name for Category",
	})

	tests := []struct {
		name     string
		body     any
		status   int                                                        // expected HTTP status
		testFunc func(t *testing.T, e controllers.EnvelopeCreateResponseV3) // tests to perform against the updated envelope resource
	}{
		{
			"Broken Body", `[{ "note": 2 }]`, http.StatusBadRequest,
			func(t *testing.T, e controllers.EnvelopeCreateResponseV3) {
				assert.Equal(t, "json: cannot unmarshal number into Go struct field EnvelopeCreateV3.note of type string", *e.Error)
			},
		},
		{"No body", "", http.StatusBadRequest, func(t *testing.T, e controllers.EnvelopeCreateResponseV3) {
			assert.Equal(t, "the request body must not be empty", *e.Error)
		}},
		{
			"No Category",
			`[{ "note": "Some text" }]`, http.StatusBadRequest,
			func(t *testing.T, e controllers.EnvelopeCreateResponseV3) {
				assert.Equal(t, "no Category ID specified", *e.Data[0].Error)
			},
		},
		{
			"Non-existing Category",
			`[{ "categoryId": "ea85ad1a-3679-4ced-b83b-89566c12ece9" }]`, http.StatusNotFound,
			func(t *testing.T, e controllers.EnvelopeCreateResponseV3) {
				assert.Equal(t, "there is no Category with this ID", *e.Data[0].Error)
			},
		},
		{
			"Duplicate name in Category",
			[]controllers.EnvelopeCreateV3{
				{
					CategoryID: e.Data.CategoryID,
					Name:       e.Data.Name,
				},
			},
			http.StatusBadRequest,
			func(t *testing.T, e controllers.EnvelopeCreateResponseV3) {
				assert.Equal(t, "the envelope name must be unique for the category", *e.Data[0].Error)
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/envelopes", tt.body)
			assertHTTPStatus(t, &r, tt.status)

			var e controllers.EnvelopeCreateResponseV3
			decodeResponse(t, &r, &e)

			if tt.testFunc != nil {
				tt.testFunc(t, e)
			}
		})
	}
}

// Verify that updating envelopes works as desired
func (suite *TestSuiteStandard) TestEnvelopesV3Update() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})

	tests := []struct {
		name     string                                               // name of the test
		envelope map[string]any                                       // the updates to perform. This is not a struct because that would set all fields on the request
		testFunc func(t *testing.T, e controllers.EnvelopeResponseV3) // tests to perform against the updated envelope resource
	}{
		{
			"Name, Note",
			map[string]any{
				"name": "Another name",
				"note": "New note!",
			},
			func(t *testing.T, e controllers.EnvelopeResponseV3) {
				assert.Equal(t, "New note!", e.Data.Note)
				assert.Equal(t, "Another name", e.Data.Name)
			},
		},
		{
			"Archived",
			map[string]any{
				"archived": true,
			},
			func(t *testing.T, e controllers.EnvelopeResponseV3) {
				assert.True(t, e.Data.Archived)
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPatch, envelope.Data.Links.Self, tt.envelope)
			assertHTTPStatus(t, &r, http.StatusOK)

			var e controllers.EnvelopeResponseV3
			suite.decodeResponse(&r, &e)

			if tt.testFunc != nil {
				tt.testFunc(t, e)
			}
		})
	}
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
		{"Non-existing Envelope", uuid.New().String(), `{"name": 2}`, http.StatusNotFound},
		{"Set Category to uuid.Nil", "", controllers.EnvelopeCreateV3{}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
					Name: "New Envelope",
					Note: "Auto-created for test",
				})

				tt.id = envelope.Data.ID.String()
			}

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
				e := suite.createTestEnvelopeV3(t, controllers.EnvelopeCreateV3{})
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
	e1 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
		Name: "Alphabetically first",
	})

	e2 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
		Name: "Second in creation, third in list",
	})

	e3 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
		Name: "First is alphabetically second",
	})

	e4 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{
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
		suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{Name: fmt.Sprint(i)})
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