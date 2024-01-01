package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v4/pkg/controllers"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestBudgetV3(t *testing.T, c models.BudgetCreate, expectedStatus ...int) controllers.BudgetResponseV3 {
	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	body := []models.BudgetCreate{
		c,
	}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/budgets", body)
	assertHTTPStatus(t, &r, expectedStatus...)

	var a controllers.BudgetCreateResponseV3
	suite.decodeResponse(&r, &a)

	if r.Code == http.StatusCreated {
		return a.Data[0]
	}

	return controllers.BudgetResponseV3{}
}

// TestBudgetsV3DBClosed verifies that errors are processed correctly when
// the database is closed.
func (suite *TestSuiteStandard) TestBudgetsV3DBClosed() {
	tests := []struct {
		name string             // Name of the test
		test func(t *testing.T) // Code to run
	}{
		{
			"Creation fails",
			func(t *testing.T) {
				suite.createTestBudgetV3(t, models.BudgetCreate{}, http.StatusInternalServerError)
			},
		},
		{
			"GET fails",
			func(t *testing.T) {
				recorder := test.Request(suite.controller, t, http.MethodGet, "http://example.com/v3/budgets", "")
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

// TestBudgetV3Options verifies that OPTIONS requests are handled correctly.
func (suite *TestSuiteStandard) TestBudgetV3Options() {
	tests := []struct {
		name   string
		id     string // path at the /v3/budgets endpoint to test
		status int    // Expected HTTP status code
	}{
		{"No budget with this ID", uuid.New().String(), http.StatusNotFound},
		{"Not a valid UUID", "NotParseableAsUUID", http.StatusBadRequest},
		{"Budget exists", suite.createTestBudgetV3(suite.T(), models.BudgetCreate{}).Data.ID.String(), http.StatusNoContent},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", "http://example.com/v3/budgets", tt.id)
			r := test.Request(suite.controller, t, http.MethodOptions, path, "")
			assertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestBudgetsV3GetSingle verifies that requests for the resource endpoints are
// handled correctly.
func (suite *TestSuiteStandard) TestBudgetsV3GetSingle() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	tests := []struct {
		name   string
		id     string
		status int
		method string
	}{
		{"GET Existing budget", budget.Data.ID.String(), http.StatusOK, http.MethodGet},
		{"GET ID nil", uuid.Nil.String(), http.StatusBadRequest, http.MethodGet},
		{"GET No budget with this ID", uuid.New().String(), http.StatusNotFound, http.MethodGet},
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
			r := test.Request(suite.controller, t, tt.method, fmt.Sprintf("http://example.com/v3/budgets/%s", tt.id), "")

			var budget controllers.BudgetResponseV3
			suite.decodeResponse(&r, &budget)
			assertHTTPStatus(t, &r, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestBudgetsV3GetFilter() {
	_ = suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
		Name:     "Exact String Match",
		Note:     "This is a specific note",
		Currency: "",
	})

	_ = suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
		Name:     "",
		Note:     "This is a specific note",
		Currency: "$",
	})

	_ = suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
		Name:     "Another String",
		Note:     "A different note",
		Currency: "€",
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Currency: €", "currency=€", 1},
		{"Currency: $", "currency=$", 1},
		{"Currency & Name", "currency=€&name=Another String", 1},
		{"Note", "note=This is a specific note", 2},
		{"Name", "name=Exact String Match", 1},
		{"Empty Name with Note", "name=&note=This is a specific note", 1},
		{"No currency", "currency=", 1},
		{"No name", "name=", 1},
		{"Search for 'stRing'", "search=stRing", 2},
		{"Search for 'Note'", "search=Note", 3},
		{"Offset", "offset=1", 2},
		{"Limit", "limit=1", 1},
	}

	var re controllers.BudgetListResponseV3
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("http://example.com/v3/budgets?%s", tt.query), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)
			suite.decodeResponse(&r, &re)
			assert.Equal(t, tt.len, len(re.Data))
		})
	}
}

func (suite *TestSuiteStandard) TestBudgetsV3CreateFails() {
	tests := []struct {
		name     string
		body     string
		testFunc func(t *testing.T, b controllers.BudgetCreateResponseV3) // tests to perform against the updated budget resource
	}{
		{"Broken Body", `{ "note": 2 }`, func(t *testing.T, b controllers.BudgetCreateResponseV3) {
			assert.Equal(t, "json: cannot unmarshal object into Go value of type []models.BudgetCreate", *b.Error)
		}},
		{"No body", "", func(t *testing.T, b controllers.BudgetCreateResponseV3) {
			assert.Equal(t, "the request body must not be empty", *b.Error)
		}},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/budgets", tt.body)
			assertHTTPStatus(t, &r, http.StatusBadRequest)

			var b controllers.BudgetCreateResponseV3
			suite.decodeResponse(&r, &b)

			if tt.testFunc != nil {
				tt.testFunc(t, b)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestBudgetsV3Update() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
		Name: "New Budget",
		Note: "More tests something something",
	})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, budget.Data.Links.Self, map[string]any{
		"name": "Updated new budget",
		"note": "",
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedBudget controllers.BudgetResponseV3
	suite.decodeResponse(&recorder, &updatedBudget)

	assert.Equal(suite.T(), "", updatedBudget.Data.Note)
	assert.Equal(suite.T(), "Updated new budget", updatedBudget.Data.Name)
}

func (suite *TestSuiteStandard) TestBudgetsV3UpdateFails() {
	tests := []struct {
		name   string
		id     string
		body   string
		status int // expected response status
	}{
		{"Invalid type", "", `{"name": 2}`, http.StatusBadRequest},
		{"Non-existing budget", uuid.New().String(), `{"name": 2}`, http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				// Create test budget
				budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
					Name: "New Budget",
					Note: "More tests something something",
				})

				tt.id = budget.Data.ID.String()
			}

			// Update budget
			recorder = test.Request(suite.controller, t, http.MethodPatch, fmt.Sprintf("http://example.com/v3/budgets/%s", tt.id), tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestBudgetsV3Delete verifies all cases for budget deletions.
func (suite *TestSuiteStandard) TestBudgetsV3Delete() {
	tests := []struct {
		name   string
		id     string
		status int // expected response status
	}{
		{"Success", "", http.StatusNoContent},
		{"Non-existing budget", uuid.New().String(), http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				// Create test budget
				b := suite.createTestBudgetV3(t, models.BudgetCreate{})
				tt.id = b.Data.ID.String()
			}

			// Update budget
			recorder = test.Request(suite.controller, t, http.MethodDelete, fmt.Sprintf("http://example.com/v3/budgets/%s", tt.id), "")
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestBudgetsV3GetSorted verifies that budgets are sorted by name.
func (suite *TestSuiteStandard) TestBudgetsV3GetSorted() {
	b1 := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
		Name: "Alphabetically first",
	})

	b2 := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
		Name: "Second in creation, third in list",
	})

	b3 := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
		Name: "First is alphabetically second",
	})

	b4 := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{
		Name: "Zulu is the last one",
	})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v3/budgets", "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)

	var budgets controllers.BudgetListResponseV3
	suite.decodeResponse(&r, &budgets)

	if !assert.Len(suite.T(), budgets.Data, 4) {
		assert.FailNow(suite.T(), "Budgets list has wrong length")
	}

	assert.Equal(suite.T(), b1.Data.Name, budgets.Data[0].Name)
	assert.Equal(suite.T(), b2.Data.Name, budgets.Data[2].Name)
	assert.Equal(suite.T(), b3.Data.Name, budgets.Data[1].Name)
	assert.Equal(suite.T(), b4.Data.Name, budgets.Data[3].Name)
}

func (suite *TestSuiteStandard) TestBudgetsV3Pagination() {
	for i := 0; i < 10; i++ {
		suite.createTestBudgetV3(suite.T(), models.BudgetCreate{Name: fmt.Sprint(i)})
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
			r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v3/budgets?offset=%d&limit=%d", tt.offset, tt.limit), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)

			var budgets controllers.BudgetListResponseV3
			suite.decodeResponse(&r, &budgets)

			assert.Equal(suite.T(), tt.offset, budgets.Pagination.Offset)
			assert.Equal(suite.T(), tt.limit, budgets.Pagination.Limit)
			assert.Equal(suite.T(), tt.expectedCount, budgets.Pagination.Count)
			assert.Equal(suite.T(), tt.expectedTotal, budgets.Pagination.Total)
		})
	}
}
