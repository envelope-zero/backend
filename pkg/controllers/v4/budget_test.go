package v4_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	v4 "github.com/envelope-zero/backend/v4/pkg/controllers/v4"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestBudget(t *testing.T, c v4.BudgetEditable, expectedStatus ...int) v4.BudgetResponse {
	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	body := []v4.BudgetEditable{
		c,
	}

	r := test.Request(t, http.MethodPost, "http://example.com/v4/budgets", body)
	test.AssertHTTPStatus(t, &r, expectedStatus...)

	var a v4.BudgetCreateResponse
	test.DecodeResponse(t, &r, &a)

	if r.Code == http.StatusCreated {
		return a.Data[0]
	}

	return v4.BudgetResponse{}
}

// TestBudgetsDBClosed verifies that errors are processed correctly when
// the database is closed.
func (suite *TestSuiteStandard) TestBudgetsDBClosed() {
	tests := []struct {
		name string             // Name of the test
		test func(t *testing.T) // Code to run
	}{
		{
			"Creation fails",
			func(t *testing.T) {
				createTestBudget(t, v4.BudgetEditable{}, http.StatusInternalServerError)
			},
		},
		{
			"GET fails",
			func(t *testing.T) {
				recorder := test.Request(t, http.MethodGet, "http://example.com/v4/budgets", "")
				test.AssertHTTPStatus(t, &recorder, http.StatusInternalServerError)
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

// TestBudgetOptions verifies that OPTIONS requests are handled correctly.
func (suite *TestSuiteStandard) TestBudgetOptions() {
	tests := []struct {
		name   string
		id     string // path at the /v4/budgets endpoint to test
		status int    // Expected HTTP status code
	}{
		{"No budget with this ID", uuid.New().String(), http.StatusNotFound},
		{"Not a valid UUID", "NotParseableAsUUID", http.StatusBadRequest},
		{"Budget exists", createTestBudget(suite.T(), v4.BudgetEditable{}).Data.ID.String(), http.StatusNoContent},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", "http://example.com/v4/budgets", tt.id)
			r := test.Request(t, http.MethodOptions, path, "")
			test.AssertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestBudgetsGetSingle verifies that requests for the resource endpoints are
// handled correctly.
func (suite *TestSuiteStandard) TestBudgetsGetSingle() {
	budget := createTestBudget(suite.T(), v4.BudgetEditable{})

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
			r := test.Request(t, tt.method, fmt.Sprintf("http://example.com/v4/budgets/%s", tt.id), "")

			var budget v4.BudgetResponse
			test.DecodeResponse(t, &r, &budget)
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestBudgetsGetFilter() {
	_ = createTestBudget(suite.T(), v4.BudgetEditable{
		Name:     "Exact String Match",
		Note:     "This is a specific note",
		Currency: "",
	})

	_ = createTestBudget(suite.T(), v4.BudgetEditable{
		Name:     "",
		Note:     "This is a specific note",
		Currency: "$",
	})

	_ = createTestBudget(suite.T(), v4.BudgetEditable{
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

	var re v4.BudgetListResponse
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodGet, fmt.Sprintf("http://example.com/v4/budgets?%s", tt.query), "")
			test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
			test.DecodeResponse(t, &r, &re)
			assert.Equal(t, tt.len, len(re.Data))
		})
	}
}

func (suite *TestSuiteStandard) TestBudgetsCreateFails() {
	tests := []struct {
		name     string
		body     string
		testFunc func(t *testing.T, b v4.BudgetCreateResponse) // tests to perform against the updated budget resource
	}{
		{"Broken Body", `{ "note": 2 }`, func(t *testing.T, b v4.BudgetCreateResponse) {
			assert.Equal(t, "json: cannot unmarshal object into Go value of type []v4.BudgetEditable", *b.Error)
		}},
		{"No body", "", func(t *testing.T, b v4.BudgetCreateResponse) {
			assert.Equal(t, "the request body must not be empty", *b.Error)
		}},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPost, "http://example.com/v4/budgets", tt.body)
			test.AssertHTTPStatus(t, &r, http.StatusBadRequest)

			var b v4.BudgetCreateResponse
			test.DecodeResponse(t, &r, &b)

			if tt.testFunc != nil {
				tt.testFunc(t, b)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestBudgetsUpdate() {
	budget := createTestBudget(suite.T(), v4.BudgetEditable{
		Name: "New Budget",
		Note: "More tests something something",
	})

	recorder := test.Request(suite.T(), http.MethodPatch, budget.Data.Links.Self, map[string]any{
		"name": "Updated new budget",
		"note": "",
	})
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedBudget v4.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedBudget)

	assert.Equal(suite.T(), "", updatedBudget.Data.Note)
	assert.Equal(suite.T(), "Updated new budget", updatedBudget.Data.Name)
}

func (suite *TestSuiteStandard) TestBudgetsUpdateFails() {
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
				budget := createTestBudget(suite.T(), v4.BudgetEditable{
					Name: "New Budget",
					Note: "More tests something something",
				})

				tt.id = budget.Data.ID.String()
			}

			// Update budget
			recorder = test.Request(t, http.MethodPatch, fmt.Sprintf("http://example.com/v4/budgets/%s", tt.id), tt.body)
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestBudgetsDelete verifies all cases for budget deletions.
func (suite *TestSuiteStandard) TestBudgetsDelete() {
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
				b := createTestBudget(t, v4.BudgetEditable{})
				tt.id = b.Data.ID.String()
			}

			// Update budget
			recorder = test.Request(t, http.MethodDelete, fmt.Sprintf("http://example.com/v4/budgets/%s", tt.id), "")
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestBudgetsGetSorted verifies that budgets are sorted by name.
func (suite *TestSuiteStandard) TestBudgetsGetSorted() {
	b1 := createTestBudget(suite.T(), v4.BudgetEditable{
		Name: "Alphabetically first",
	})

	b2 := createTestBudget(suite.T(), v4.BudgetEditable{
		Name: "Second in creation, third in list",
	})

	b3 := createTestBudget(suite.T(), v4.BudgetEditable{
		Name: "First is alphabetically second",
	})

	b4 := createTestBudget(suite.T(), v4.BudgetEditable{
		Name: "Zulu is the last one",
	})

	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v4/budgets", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)

	var budgets v4.BudgetListResponse
	test.DecodeResponse(suite.T(), &r, &budgets)

	require.Len(suite.T(), budgets.Data, 4, "Budgets list has wrong length")

	assert.Equal(suite.T(), b1.Data.Name, budgets.Data[0].Name)
	assert.Equal(suite.T(), b2.Data.Name, budgets.Data[2].Name)
	assert.Equal(suite.T(), b3.Data.Name, budgets.Data[1].Name)
	assert.Equal(suite.T(), b4.Data.Name, budgets.Data[3].Name)
}

func (suite *TestSuiteStandard) TestBudgetsPagination() {
	for i := 0; i < 10; i++ {
		createTestBudget(suite.T(), v4.BudgetEditable{Name: fmt.Sprint(i)})
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
			r := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v4/budgets?offset=%d&limit=%d", tt.offset, tt.limit), "")
			test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)

			var budgets v4.BudgetListResponse
			test.DecodeResponse(t, &r, &budgets)

			assert.Equal(suite.T(), tt.offset, budgets.Pagination.Offset)
			assert.Equal(suite.T(), tt.limit, budgets.Pagination.Limit)
			assert.Equal(suite.T(), tt.expectedCount, budgets.Pagination.Count)
			assert.Equal(suite.T(), tt.expectedTotal, budgets.Pagination.Total)
		})
	}
}
