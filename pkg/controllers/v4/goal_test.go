package v4_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v5/internal/types"
	v4 "github.com/envelope-zero/backend/v5/pkg/controllers/v4"
	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func createTestGoal(t *testing.T, c v4.GoalEditable, expectedStatus ...int) v4.GoalResponse {
	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	if c.EnvelopeID == uuid.Nil {
		c.EnvelopeID = createTestEnvelope(t, v4.EnvelopeEditable{}).Data.ID
	}

	requestBody := []v4.GoalEditable{c}

	recorder := test.Request(t, http.MethodPost, "http://example.com/v4/goals", requestBody)
	test.AssertHTTPStatus(t, &recorder, expectedStatus...)

	var response v4.GoalCreateResponse
	test.DecodeResponse(t, &recorder, &response)

	return response.Data[0]
}

// TestGoalsOptions verifies that the HTTP OPTIONS response for //goals/{id} is correct.
func (suite *TestSuiteStandard) TestGoalsOptions() {
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
				return createTestGoal(suite.T(), v4.GoalEditable{Amount: decimal.NewFromFloat(31)}).Data.Links.Self
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var p string
			if tt.pathFunc != nil {
				p = tt.pathFunc()
			} else {
				p = fmt.Sprintf("%s/%s", "http://example.com/v4/goals", tt.id)
			}

			r := test.Request(t, http.MethodOptions, p, "")
			test.AssertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestGoalsDatabaseError verifies that the endpoints return the appropriate
// error when the database is disconncted.
func (suite *TestSuiteStandard) TestGoalsDatabaseError() {
	tests := []struct {
		name   string // Name of the test
		path   string // Path to send request to
		method string // HTTP method to use
	}{
		{"GET Collection", "", http.MethodGet},
		// Skipping POST Collection here since we need to check the indivdual transactions for that one
		{"OPTIONS Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodOptions},
		{"GET Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodGet},
		{"PATCH Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodPatch},
		{"DELETE Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodDelete},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			suite.CloseDB()

			recorder := test.Request(t, tt.method, fmt.Sprintf("http://example.com/v4/goals%s", tt.path), "")
			test.AssertHTTPStatus(t, &recorder, http.StatusInternalServerError)

			var response struct {
				Error string `json:"error"`
			}
			test.DecodeResponse(t, &recorder, &response)

			assert.Equal(t, models.ErrGeneral.Error(), response.Error)
		})
	}
}

// TestGoalsGet verifies that goals can be read from the API and
// that the default sorting is correct.
func (suite *TestSuiteStandard) TestGoalsGet() {
	g1 := createTestGoal(suite.T(), v4.GoalEditable{
		Amount: decimal.NewFromFloat(100),
		Month:  types.NewMonth(2024, 1),
		Name:   "Irrelevant",
	})

	_ = createTestGoal(suite.T(), v4.GoalEditable{
		Amount: decimal.NewFromFloat(300),
		Month:  types.NewMonth(2024, 2),
		Name:   "First",
	})

	g3 := createTestGoal(suite.T(), v4.GoalEditable{
		Amount: decimal.NewFromFloat(50),
		Month:  types.NewMonth(2024, 2),
		Name:   "Before g2",
	})

	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v4/goals", "")

	var response v4.GoalListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 3)

	// Verify that the goal with the earlier month is the first in the list
	assert.Equal(suite.T(), g1.Data.ID, response.Data[0].ID)

	// Verify that the goal with the alphabetically earlier name is earlier
	assert.Equal(suite.T(), g3.Data.ID, response.Data[1].ID)
}

// TestGoalsGetFilter verifies that filtering goals works as expected.
func (suite *TestSuiteStandard) TestGoalsGetFilter() {
	b := createTestBudget(suite.T(), v4.BudgetEditable{})

	c := createTestCategory(suite.T(), v4.CategoryEditable{BudgetID: b.Data.ID})

	e1 := createTestEnvelope(suite.T(), v4.EnvelopeEditable{CategoryID: c.Data.ID})
	e2 := createTestEnvelope(suite.T(), v4.EnvelopeEditable{CategoryID: c.Data.ID})

	_ = createTestGoal(suite.T(), v4.GoalEditable{
		Name:       "Test Goal",
		Note:       "So that we can go to X",
		EnvelopeID: e1.Data.ID,
		Amount:     decimal.NewFromFloat(100),
		Month:      types.NewMonth(2024, 1),
		Archived:   false,
	})

	_ = createTestGoal(suite.T(), v4.GoalEditable{
		Name:       "Goal for something else",
		EnvelopeID: e1.Data.ID,
		Amount:     decimal.NewFromFloat(200),
		Month:      types.NewMonth(2024, 2),
		Archived:   true,
	})

	_ = createTestGoal(suite.T(), v4.GoalEditable{
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
			var re v4.GoalListResponse
			r := test.Request(t, http.MethodGet, fmt.Sprintf("/v4/goals?%s", tt.query), "")
			test.AssertHTTPStatus(t, &r, http.StatusOK)
			test.DecodeResponse(t, &r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

// TestGoalsGetInvalidQuery verifies that invalid filtering queries
// return a HTTP Bad Request.
func (suite *TestSuiteStandard) TestGoalsGetInvalidQuery() {
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
			recorder := test.Request(t, http.MethodGet, fmt.Sprintf("http://example.com/v4/goals?%s", tt), "")
			test.AssertHTTPStatus(t, &recorder, http.StatusBadRequest)
		})
	}
}

// TestGoalsCreateInvalidBody verifies that creation of goals
// with an unparseable request body returns a HTTP Bad Request.
func (suite *TestSuiteStandard) TestGoalsCreateInvalidBody() {
	r := test.Request(suite.T(), http.MethodPost, "http://example.com/v4/goals", `{ Invalid request": Body }`)
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	var response v4.GoalCreateResponse
	test.DecodeResponse(suite.T(), &r, &response)

	assert.Equal(suite.T(), httputil.ErrInvalidBody.Error(), *response.Error)
	assert.Nil(suite.T(), response.Data)
}

// TestGoalsCreate verifies that transaction goal works.
func (suite *TestSuiteStandard) TestGoalsCreate() {
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{Name: "An envelope for this test"})

	tests := []struct {
		name           string
		goals          []v4.GoalEditable
		expectedStatus int
		expectedError  *error   // Error expected in the response
		expectedErrors []string // Errors expected for the individual transactions
	}{
		{
			"One success, one fail",
			[]v4.GoalEditable{
				{
					EnvelopeID: uuid.New(),
					Amount:     decimal.NewFromFloat(17.23),
					Note:       "v4 non-existing envelope ID",
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
				"there is no envelope matching your query",
				"",
			},
		},
		{
			"Both succeed",
			[]v4.GoalEditable{
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
			r := test.Request(t, http.MethodPost, "http://example.com/v4/goals", tt.goals)
			test.AssertHTTPStatus(t, &r, tt.expectedStatus)

			var response v4.GoalCreateResponse
			test.DecodeResponse(t, &r, &response)

			for i, goal := range response.Data {
				if tt.expectedErrors[i] == "" {
					assert.Equal(t, fmt.Sprintf("http://example.com/v4/goals/%s", goal.Data.ID), goal.Data.Links.Self)
				} else {
					// This needs to be in the else to prevent nil pointer errors since we're dereferencing pointers
					assert.Equal(t, tt.expectedErrors[i], *goal.Error)
				}
			}
		})
	}
}

// TestGoalsGetSingle verifies that a goal can be read from the API via its link
// and that the link is for API v4.
func (suite *TestSuiteStandard) TestGoalsGetSingle() {
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
				return createTestGoal(suite.T(), v4.GoalEditable{Amount: decimal.NewFromFloat(42)}).Data.Links.Self
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
				p = fmt.Sprintf("%s/%s", "http://example.com/v4/goals", tt.id)
			}

			r := test.Request(t, http.MethodGet, p, "")
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestGoalsDelete verifies the correct success and error responses
// for DELETE requests.
func (suite *TestSuiteStandard) TestGoalsDelete() {
	tests := []struct {
		name   string // Name for the test
		status int    // Expected HTTP status
		id     string // String to use as ID.
	}{
		{
			"Standard deletion",
			http.StatusNoContent,
			createTestGoal(suite.T(), v4.GoalEditable{Amount: decimal.NewFromFloat(2100)}).Data.ID.String(),
		},
		{
			"Does not exist",
			http.StatusNotFound,
			"4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501",
		},
		{
			"Null transaction",
			http.StatusNotFound,
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
			p := fmt.Sprintf("%s/%s", "http://example.com/v4/goals", tt.id)

			r := test.Request(t, http.MethodDelete, p, "")
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestGoalsUpdateFail verifies that goal updates fail where they should.
func (suite *TestSuiteStandard) TestGoalsUpdateFail() {
	goal := createTestGoal(suite.T(), v4.GoalEditable{Amount: decimal.NewFromFloat(170), Note: "Test note for goal"})

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
			http.StatusNotFound,
			v4.GoalEditable{}, // Sets the EnvelopeID to uuid.Nil
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
			p := fmt.Sprintf("%s/%s", "http://example.com/v4/goals", tt.id)

			r := test.Request(t, http.MethodPatch, p, tt.body)
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestUpdateNonExistingGoal verifies that patching a non-existent transaction returns a 404.
func (suite *TestSuiteStandard) TestUpdateNonExistingGoal() {
	recorder := test.Request(suite.T(), http.MethodPatch, "http://example.com/v4/goals/c08c0a04-2a12-4cb9-8b4a-87cf270cdd8d", `{ "note": "2" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

// TestGoalsUpdate verifies that transaction updates are successful.
func (suite *TestSuiteStandard) TestGoalsUpdate() {
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{})

	goal := createTestGoal(suite.T(), v4.GoalEditable{
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
			r := test.Request(t, http.MethodPatch, goal.Data.Links.Self, tt.body)
			test.AssertHTTPStatus(t, &r, http.StatusOK)
		})
	}
}
