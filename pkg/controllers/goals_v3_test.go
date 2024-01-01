package controllers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/controllers"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestGoalV3(t *testing.T, c controllers.GoalV3Editable, expectedStatus ...int) controllers.GoalResponseV3 {
	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	if c.EnvelopeID == uuid.Nil {
		c.EnvelopeID = suite.createTestEnvelopeV3(t, controllers.EnvelopeCreateV3{}).Data.ID
	}

	requestBody := []controllers.GoalV3Editable{c}

	recorder := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/goals", requestBody)
	assertHTTPStatus(t, &recorder, expectedStatus...)

	var response controllers.GoalCreateResponseV3
	suite.decodeResponse(&recorder, &response)

	return response.Data[0]
}

// TestGoalsV3Options verifies that the HTTP OPTIONS response for /v3/goals/{id} is correct.
func (suite *TestSuiteStandard) TestGoalsV3Options() {
	tests := []struct {
		name     string        // Name for the test
		status   int           // Expected HTTP status
		id       string        // String to use as ID. Ignored when pathFunc is non-nil
		pathFunc func() string // Function returning the path
	}{
		{
			"Does not exist",
			http.StatusNotFound,
			uuid.New().String(),
			nil,
		},
		{
			"Invalid UUID",
			http.StatusBadRequest,
			"NotParseableAsUUID",
			nil,
		},
		{
			"Success",
			http.StatusNoContent,
			"",
			func() string {
				return suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{Amount: decimal.NewFromFloat(31)}).Data.Links.Self
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var p string
			if tt.pathFunc != nil {
				p = tt.pathFunc()
			} else {
				p = fmt.Sprintf("%s/%s", "http://example.com/v3/goals", tt.id)
			}

			r := test.Request(suite.controller, t, http.MethodOptions, p, "")
			assertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestGoalsV3DatabaseError verifies that the endpoints return the appropriate
// error when the database is disconncted.
func (suite *TestSuiteStandard) TestGoalsV3DatabaseError() {
	tests := []struct {
		name   string // Name of the test
		path   string // Path to send request to
		method string // HTTP method to use
		body   string // The request body
	}{
		{"GET Collection", "", http.MethodGet, ""},
		// Skipping POST Collection here since we need to check the indivdual transactions for that one
		{"OPTIONS Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodOptions, ""},
		{"GET Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodGet, ""},
		{"PATCH Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodPatch, ""},
		{"DELETE Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodDelete, ""},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			suite.CloseDB()

			recorder := test.Request(suite.controller, t, tt.method, fmt.Sprintf("http://example.com/v3/goals%s", tt.path), tt.body)
			assertHTTPStatus(t, &recorder, http.StatusInternalServerError)
			assert.Equal(t, httperrors.ErrDatabaseClosed.Error(), test.DecodeError(t, recorder.Body.Bytes()))
		})
	}
}

// TestGoalsV3Get verifies that goals can be read from the API and
// that the default sorting is correct.
func (suite *TestSuiteStandard) TestGoalsV3Get() {
	g1 := suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{
		Amount: decimal.NewFromFloat(100),
		Month:  types.NewMonth(2024, 1),
		Name:   "Irrelevant",
	})

	_ = suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{
		Amount: decimal.NewFromFloat(300),
		Month:  types.NewMonth(2024, 2),
		Name:   "First",
	})

	g3 := suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{
		Amount: decimal.NewFromFloat(50),
		Month:  types.NewMonth(2024, 2),
		Name:   "Before g2",
	})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v3/goals", "")

	var response controllers.GoalListResponseV3
	suite.decodeResponse(&recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 3)

	// Verify that the goal with the earlier month is the first in the list
	assert.Equal(suite.T(), g1.Data.ID, response.Data[0].ID)

	// Verify that the goal with the alphabetically earlier name is earlier
	assert.Equal(suite.T(), g3.Data.ID, response.Data[1].ID)
}

// TestGoalsV3GetFilter verifies that filtering goals works as expected.
func (suite *TestSuiteStandard) TestGoalsV3GetFilter() {
	b := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	c := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: b.Data.ID})

	e1 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: c.Data.ID})
	e2 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: c.Data.ID})

	_ = suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{
		Name:       "Test Goal",
		Note:       "So that we can go to X",
		EnvelopeID: e1.Data.ID,
		Amount:     decimal.NewFromFloat(100),
		Month:      types.NewMonth(2024, 1),
		Archived:   false,
	})

	_ = suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{
		Name:       "Goal for something else",
		EnvelopeID: e1.Data.ID,
		Amount:     decimal.NewFromFloat(200),
		Month:      types.NewMonth(2024, 2),
		Archived:   true,
	})

	_ = suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{
		Name:       "testing the filters",
		Note:       "so that I know they work",
		EnvelopeID: e2.Data.ID,
		Amount:     decimal.NewFromFloat(1000),
		Month:      types.NewMonth(2024, 1),
		Archived:   false,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Same month", fmt.Sprintf("month=%s", types.NewMonth(2024, 1)), 2},
		{"After month", fmt.Sprintf("fromMonth=%s", types.NewMonth(2024, 2)), 1},
		{"Before month", fmt.Sprintf("untilMonth=%s", types.NewMonth(2024, 2)), 3},
		{"After all months", fmt.Sprintf("fromMonth=%s", types.NewMonth(2024, 6)), 0},
		{"Before all months", fmt.Sprintf("untilMonth=%s", types.NewMonth(2023, 6)), 0},
		{"Impossible between two months", fmt.Sprintf("fromMonth=%s&untilMonth=%s", types.NewMonth(2024, 11), types.NewMonth(2024, 10)), 0},
		{"Exact Amount", fmt.Sprintf("amount=%s", decimal.NewFromFloat(200).String()), 1},
		{"Note", "note=can", 1},
		{"No note", "note=", 1},
		{"Fuzzy note", "note=so", 2},
		{"Amount less or equal to 99", "amountLessOrEqual=99", 0},
		{"Amount less or equal to 200", "amountLessOrEqual=200", 2},
		{"Amount more or equal to 3", "amountMoreOrEqual=3", 3},
		{"Amount more or equal to 500.813", "amountMoreOrEqual=500.813", 1},
		{"Amount more or equal to 99999", "amountMoreOrEqual=99999", 0},
		{"Amount more or equal to 100 and less than 10", "amountMoreOrEqual=100&amountLessOrEqual=10", 0},
		{"Amount more or equal to 50 and less than 500", "amountMoreOrEqual=50&amountLessOrEqual=500", 2},
		{"Limit positive", "limit=2", 2},
		{"Limit zero", "limit=0", 0},
		{"Limit unset", "limit=-1", 3},
		{"Limit negative", "limit=-123", 3},
		{"Offset zero", "offset=0", 3},
		{"Offset positive", "offset=2", 1},
		{"Offset higher than number", "offset=5", 0},
		{"Limit and Offset", "limit=1&offset=1", 1},
		{"Limit and Fuzzy Note", "limit=1&note=so", 1},
		{"Offset and Fuzzy Note", "offset=2&note=they", 0},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.GoalListResponseV3
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v3/goals?%s", tt.query), "")
			assertHTTPStatus(t, &r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

// TestGoalsV3GetInvalidQuery verifies that invalid filtering queries
// return a HTTP Bad Request.
func (suite *TestSuiteStandard) TestGoalsV3GetInvalidQuery() {
	tests := []string{
		"envelope=ThisIsDefinitelyACat!",
		"month=A long time ago",
		"archived=0.00",
		"amount=Seventeen Cents",
		"offset=-1",             // offset is a uint
		"limit=name",            // limit is an int
		"untilMonth=2023-11-01", // Format is "YYYY-MM"
		"fromMonth=Yesterday",
	}

	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("http://example.com/v3/goals?%s", tt), "")
			assertHTTPStatus(t, &recorder, http.StatusBadRequest)
		})
	}
}

// TestGoalsV3CreateInvalidBody verifies that creation of goals
// with an unparseable request body returns a HTTP Bad Request.
func (suite *TestSuiteStandard) TestGoalsV3CreateInvalidBody() {
	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v3/goals", `{ Invalid request": Body }`)
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	var response controllers.GoalCreateResponseV3
	suite.decodeResponse(&r, &response)

	assert.Equal(suite.T(), httperrors.ErrInvalidBody.Error(), *response.Error)
	assert.Nil(suite.T(), response.Data)
}

// TestGoalsV3Create verifies that transaction goal works.
func (suite *TestSuiteStandard) TestGoalsV3Create() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{Name: "An envelope for this test"})

	tests := []struct {
		name           string
		goals          []controllers.GoalV3Editable
		expectedStatus int
		expectedError  *error   // Error expected in the response
		expectedErrors []string // Errors expected for the individual transactions
	}{
		{
			"One success, one fail",
			[]controllers.GoalV3Editable{
				{
					EnvelopeID: uuid.New(),
					Amount:     decimal.NewFromFloat(17.23),
					Note:       "v3 non-existing envelope ID",
					Name:       "One success, one fail",
				},
				{
					EnvelopeID: envelope.Data.ID,
					Amount:     decimal.NewFromFloat(57.01),
				},
			},
			http.StatusNotFound,
			nil,
			[]string{
				"there is no Envelope with this ID",
				"",
			},
		},
		{
			"Both succeed",
			[]controllers.GoalV3Editable{
				{
					Name:       "Both succeed - 1",
					EnvelopeID: envelope.Data.ID,
					Amount:     decimal.NewFromFloat(17.23),
				},
				{
					Name:       "Unique Name for the Envelope",
					EnvelopeID: envelope.Data.ID,
					Amount:     decimal.NewFromFloat(57.01),
				},
			},
			http.StatusCreated,
			nil,
			[]string{
				"",
				"",
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/goals", tt.goals)
			assertHTTPStatus(t, &r, tt.expectedStatus)

			var response controllers.GoalCreateResponseV3
			suite.decodeResponse(&r, &response)

			for i, goal := range response.Data {
				if tt.expectedErrors[i] == "" {
					assert.Equal(t, fmt.Sprintf("http://example.com/v3/goals/%s", goal.Data.ID), goal.Data.Links.Self)
				} else {
					// This needs to be in the else to prevent nil pointer errors since we're dereferencing pointers
					assert.Equal(t, tt.expectedErrors[i], *goal.Error)
				}
			}
		})
	}
}

// TestGoalsV3GetSingle verifies that a goal can be read from the API via its link
// and that the link is for API v3.
func (suite *TestSuiteStandard) TestGoalsV3GetSingle() {
	tests := []struct {
		name     string        // Name for the test
		status   int           // Expected HTTP status
		id       string        // String to use as ID. Ignored when pathFunc is non-nil
		pathFunc func() string // Function returning the path
	}{
		{
			"Standard transaction",
			http.StatusOK,
			"",
			func() string {
				return suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{Amount: decimal.NewFromFloat(42)}).Data.Links.Self
			},
		},
		{
			"Invalid UUID",
			http.StatusBadRequest,
			"NotParseableAsUUID",
			nil,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var p string
			if tt.pathFunc != nil {
				p = tt.pathFunc()
			} else {
				p = fmt.Sprintf("%s/%s", "http://example.com/v3/goals", tt.id)
			}

			r := test.Request(suite.controller, suite.T(), http.MethodGet, p, "")
			assertHTTPStatus(suite.T(), &r, tt.status)
		})
	}
}

// TestGoalsV3Delete verifies the correct success and error responses
// for DELETE requests.
func (suite *TestSuiteStandard) TestGoalsV3Delete() {
	tests := []struct {
		name   string // Name for the test
		status int    // Expected HTTP status
		id     string // String to use as ID.
	}{
		{
			"Standard deletion",
			http.StatusNoContent,
			suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{Amount: decimal.NewFromFloat(2100)}).Data.ID.String(),
		},
		{
			"Does not exist",
			http.StatusNotFound,
			"4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501",
		},
		{
			"Null transaction",
			http.StatusBadRequest,
			"00000000-0000-0000-0000-000000000000",
		},
		{
			"Invalid UUID",
			http.StatusBadRequest,
			"Definitely an Invalid ID",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s", "http://example.com/v3/goals", tt.id)

			r := test.Request(suite.controller, t, http.MethodDelete, p, "")
			assertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestGoalsV3UpdateFail verifies that goal updates fail where they should.
func (suite *TestSuiteStandard) TestGoalsV3UpdateFail() {
	goal := suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{Amount: decimal.NewFromFloat(170), Note: "Test note for goal"})

	tests := []struct {
		name   string // Name for the test
		id     string // ID of the Goal to update
		status int    // Expected HTTP status
		body   any    // Body to send to the PATCH endpoint
	}{
		{
			"Invalid body",
			goal.Data.ID.String(),
			http.StatusBadRequest,
			`{ "amount": 2" }`,
		},
		{
			"Invalid type",
			goal.Data.ID.String(),
			http.StatusBadRequest,
			map[string]any{
				"amount": false,
			},
		},
		{
			"Invalid goal ID",
			"Not a valid UUID",
			http.StatusBadRequest,
			``,
		},
		{
			"Invalid envelope ID",
			goal.Data.ID.String(),
			http.StatusBadRequest,
			controllers.GoalV3Editable{}, // Sets the EnvelopeID to uuid.Nil
		},
		{
			"Negative amount",
			goal.Data.ID.String(),
			http.StatusBadRequest,
			`{ "amount": -58.23 }`,
		},
		{
			"Non-existing envelope",
			goal.Data.ID.String(),
			http.StatusNotFound,
			`{ "envelopeId": "e6fa8eb5-5f2c-4292-8ef9-02f0c2af1ce4" }`,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s", "http://example.com/v3/goals", tt.id)

			r := test.Request(suite.controller, t, http.MethodPatch, p, tt.body)
			assertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestUpdateNonExistingGoalV3 verifies that patching a non-existent transaction returns a 404.
func (suite *TestSuiteStandard) TestUpdateNonExistingGoalV3() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v3/goals/c08c0a04-2a12-4cb9-8b4a-87cf270cdd8d", `{ "note": "2" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

// TestGoalsV3Update verifies that transaction updates are successful.
func (suite *TestSuiteStandard) TestGoalsV3Update() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})

	goal := suite.createTestGoalV3(suite.T(), controllers.GoalV3Editable{
		Amount:     decimal.NewFromFloat(23.14),
		Note:       "Test note for transaction",
		Archived:   true,
		EnvelopeID: envelope.Data.ID,
	})

	tests := []struct {
		name string // Name for the test
		body any    // Body to send to the PATCH endpoint
	}{
		{
			"Empty note",
			map[string]any{
				"note": "",
			},
		},
		{
			"Change amount",
			map[string]any{
				"amount": decimal.NewFromFloat(130),
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPatch, goal.Data.Links.Self, tt.body)
			assertHTTPStatus(t, &r, http.StatusOK)
		})
	}
}
