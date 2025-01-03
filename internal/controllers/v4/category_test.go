package v4_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	v4 "github.com/envelope-zero/backend/v5/internal/controllers/v4"
	"github.com/envelope-zero/backend/v5/internal/models"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCategory(t *testing.T, c v4.CategoryEditable, expectedStatus ...int) v4.CategoryResponse {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = createTestBudget(t, v4.BudgetEditable{Name: "Testing budget"}).Data.ID
	}

	if c.Name == "" {
		c.Name = uuid.NewString()
	}

	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	body := []v4.CategoryEditable{c}

	r := test.Request(t, http.MethodPost, "http://example.com/v4/categories", body)
	test.AssertHTTPStatus(t, &r, expectedStatus...)

	var category v4.CategoryCreateResponse
	test.DecodeResponse(t, &r, &category)

	if r.Code == http.StatusCreated {
		return category.Data[0]
	}

	return v4.CategoryResponse{}
}

// TestCategoriesDBClosed verifies that errors are processed correctly when
// the database is closed.
func (suite *TestSuiteStandard) TestCategoriesDBClosed() {
	b := createTestBudget(suite.T(), v4.BudgetEditable{})

	tests := []struct {
		name string             // Name of the test
		test func(t *testing.T) // Code to run
	}{
		{
			"Creation fails",
			func(t *testing.T) {
				createTestCategory(t, v4.CategoryEditable{BudgetID: b.Data.ID}, http.StatusInternalServerError)
			},
		},
		{
			"GET fails",
			func(t *testing.T) {
				recorder := test.Request(t, http.MethodGet, "http://example.com/v4/categories", "")
				test.AssertHTTPStatus(t, &recorder, http.StatusInternalServerError)

				var response v4.CategoryListResponse
				test.DecodeResponse(t, &recorder, &response)
				assert.Contains(t, *response.Error, models.ErrGeneral.Error())
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

// TestCategoriesOptions verifies that OPTIONS requests are handled correctly.
func (suite *TestSuiteStandard) TestCategoriesOptions() {
	tests := []struct {
		name   string
		id     string // path at the Accounts endpoint to test
		status int    // Expected HTTP status code
	}{
		{"No Category with this ID", uuid.New().String(), http.StatusNotFound},
		{"Not a valid UUID", "NotParseableAsUUID", http.StatusBadRequest},
		{"Category exists", createTestCategory(suite.T(), v4.CategoryEditable{}).Data.ID.String(), http.StatusNoContent},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", "http://example.com/v4/categories", tt.id)
			r := test.Request(t, http.MethodOptions, path, "")
			test.AssertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestCategoriesGetSingle verifies that requests for the resource endpoints are
// handled correctly.
func (suite *TestSuiteStandard) TestCategoriesGetSingle() {
	c := createTestCategory(suite.T(), v4.CategoryEditable{})

	tests := []struct {
		name   string
		id     string
		status int
		method string
	}{
		{"GET Existing Category", c.Data.ID.String(), http.StatusOK, http.MethodGet},
		{"GET ID nil", uuid.Nil.String(), http.StatusNotFound, http.MethodGet},
		{"GET No Category with this ID", uuid.New().String(), http.StatusNotFound, http.MethodGet},
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
			r := test.Request(t, tt.method, fmt.Sprintf("http://example.com/v4/categories/%s", tt.id), "")

			var category v4.CategoryResponse
			test.DecodeResponse(t, &r, &category)
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestCategoriesGetFilter() {
	b1 := createTestBudget(suite.T(), v4.BudgetEditable{})
	b2 := createTestBudget(suite.T(), v4.BudgetEditable{})

	_ = createTestCategory(suite.T(), v4.CategoryEditable{
		Name:     "Category Name",
		Note:     "A note for this category",
		BudgetID: b1.Data.ID,
		Archived: true,
	})

	_ = createTestCategory(suite.T(), v4.CategoryEditable{
		Name:     "Groceries",
		Note:     "For Groceries",
		BudgetID: b2.Data.ID,
	})

	_ = createTestCategory(suite.T(), v4.CategoryEditable{
		Name:     "Daily stuff",
		Note:     "Groceries, Drug Store, …",
		BudgetID: b2.Data.ID,
	})

	tests := []struct {
		name      string
		query     string
		len       int
		checkFunc func(t *testing.T, accounts []v4.Category)
	}{
		{"Budget 1", fmt.Sprintf("budget=%s", b1.Data.ID), 1, nil},
		{"Budget Not Existing", "budget=c9e4ee7a-e702-4f92-b168-11a95b22c7aa", 0, nil},
		{"Empty Note", "note=", 0, nil},
		{"Empty Name", "name=", 0, nil},
		{"Name & Note", "name=Category Name&note=A note for this category", 1, nil},
		{"Fuzzy name, no note", "name=Category&note=", 0, nil},
		{"Fuzzy name", "name=t", 2, nil},
		{"Fuzzy note, no name", "name=&note=Groceries", 0, nil},
		{"Fuzzy note", "note=Groceries", 2, nil},
		{"Not archived", "archived=false", 2, func(t *testing.T, categories []v4.Category) {
			for _, c := range categories {
				assert.False(t, c.Archived)
			}
		}},
		{"Archived", "archived=true", 1, func(t *testing.T, categories []v4.Category) {
			for _, c := range categories {
				assert.True(t, c.Archived)
			}
		}},
		{"Search for 'groceries'", "search=groceries", 2, nil},
		{"Search for 'FOR'", "search=FOR", 2, nil},
		{"Offset 2", "offset=2", 1, nil},
		{"Offset 0, limit 2", "offset=0&limit=2", 2, nil},
		{"Limit 4", "limit=4", 3, nil},
		{"Limit 0", "limit=0", 0, nil},
		{"Limit -1", "limit=-1", 3, nil},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re v4.CategoryListResponse
			r := test.Request(t, http.MethodGet, fmt.Sprintf("/v4/categories?%s", tt.query), "")
			test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
			test.DecodeResponse(t, &r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestCategoriesCreateFails() {
	// Test category for uniqueness
	c := createTestCategory(suite.T(), v4.CategoryEditable{
		Name: "Unique Category Name for Budget",
	})

	tests := []struct {
		name     string
		body     any
		status   int                                             // expected HTTP status
		testFunc func(t *testing.T, c v4.CategoryCreateResponse) // tests to perform against the updated category resource
	}{
		{
			"Broken Body", `[{ "note": 2 }]`, http.StatusBadRequest,
			func(t *testing.T, c v4.CategoryCreateResponse) {
				assert.Equal(t, "json: cannot unmarshal number into Go struct field CategoryEditable.note of type string", *c.Error)
			},
		},
		{
			"No body", "", http.StatusBadRequest,
			func(t *testing.T, c v4.CategoryCreateResponse) {
				assert.Equal(t, "the request body must not be empty", *c.Error)
			},
		},
		{
			"No Budget",
			`[{ "note": "Some text" }]`,
			http.StatusNotFound,
			func(t *testing.T, c v4.CategoryCreateResponse) {
				assert.Equal(t, "there is no budget matching your query", *c.Data[0].Error)
			},
		},
		{
			"Non-existing Budget",
			`[{ "budgetId": "ea85ad1a-3679-4ced-b83b-89566c12ece9" }]`,
			http.StatusNotFound,
			func(t *testing.T, c v4.CategoryCreateResponse) {
				assert.Equal(t, "there is no budget matching your query", *c.Data[0].Error)
			},
		},
		{
			"Duplicate name in Budget",
			[]v4.CategoryEditable{
				{
					BudgetID: c.Data.BudgetID,
					Name:     c.Data.Name,
				},
			},
			http.StatusBadRequest,
			func(t *testing.T, c v4.CategoryCreateResponse) {
				assert.Equal(t, "the category name must be unique for the budget", *c.Data[0].Error)
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPost, "http://example.com/v4/categories", tt.body)
			test.AssertHTTPStatus(t, &r, tt.status)

			var c v4.CategoryCreateResponse
			test.DecodeResponse(t, &r, &c)

			if tt.testFunc != nil {
				tt.testFunc(t, c)
			}
		})
	}
}

// Verify that updating categories works as desired
func (suite *TestSuiteStandard) TestCategoriesUpdate() {
	budget := createTestBudget(suite.T(), v4.BudgetEditable{})
	category := createTestCategory(suite.T(), v4.CategoryEditable{Name: "Name of the category", BudgetID: budget.Data.ID})

	tests := []struct {
		name     string                                    // name of the test
		category map[string]any                            // the updates to perform. This is not a struct because that would set all fields on the request
		testFunc func(t *testing.T, a v4.CategoryResponse) // tests to perform against the updated category resource
	}{
		{
			"Name, Note",
			map[string]any{
				"name": "Another name",
				"note": "New note!",
			},
			func(t *testing.T, a v4.CategoryResponse) {
				assert.Equal(t, "New note!", a.Data.Note)
				assert.Equal(t, "Another name", a.Data.Name)
			},
		},
		{
			"Archived",
			map[string]any{
				"archived": true,
			},
			func(t *testing.T, a v4.CategoryResponse) {
				assert.True(t, a.Data.Archived)
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPatch, category.Data.Links.Self, tt.category)
			test.AssertHTTPStatus(t, &r, http.StatusOK)

			var c v4.CategoryResponse
			test.DecodeResponse(t, &r, &c)

			if tt.testFunc != nil {
				tt.testFunc(t, c)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestCategoriesUpdateFails() {
	tests := []struct {
		name   string
		id     string
		body   any
		status int // expected response status
	}{
		{"Invalid type", "", `{"name": 2}`, http.StatusBadRequest},
		{"Broken JSON", "", `{ "name": 2" }`, http.StatusBadRequest},
		{"Non-existing Category", uuid.New().String(), `{"name": 2}`, http.StatusNotFound},
		{"Set Budget to uuid.Nil", "", v4.CategoryEditable{}, http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				envelope := createTestCategory(suite.T(), v4.CategoryEditable{
					Name: "New Envelope",
					Note: "Auto-created for test",
				})

				tt.id = envelope.Data.ID.String()
			}

			recorder = test.Request(t, http.MethodPatch, fmt.Sprintf("http://example.com/v4/categories/%s", tt.id), tt.body)
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestCategoriesDelete verifies all cases for Account deletions.
func (suite *TestSuiteStandard) TestCategoriesDelete() {
	tests := []struct {
		name   string
		id     string
		status int // expected response status
	}{
		{"Success", "", http.StatusNoContent},
		{"Non-existing Category", uuid.New().String(), http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				// Create test Account
				e := createTestCategory(t, v4.CategoryEditable{})
				tt.id = e.Data.ID.String()
			}

			// Delete Account
			recorder = test.Request(t, http.MethodDelete, fmt.Sprintf("http://example.com/v4/categories/%s", tt.id), "")
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestCategoriesGetSorted verifies that Accounts are sorted by name.
func (suite *TestSuiteStandard) TestCategoriesGetSorted() {
	c1 := createTestCategory(suite.T(), v4.CategoryEditable{
		Name: "Alphabetically first",
	})

	c2 := createTestCategory(suite.T(), v4.CategoryEditable{
		Name: "Second in creation, third in list",
	})

	c3 := createTestCategory(suite.T(), v4.CategoryEditable{
		Name: "First is alphabetically second",
	})

	c4 := createTestCategory(suite.T(), v4.CategoryEditable{
		Name: "Zulu is the last one",
	})

	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v4/categories", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)

	var categories v4.CategoryListResponse
	test.DecodeResponse(suite.T(), &r, &categories)

	require.Len(suite.T(), categories.Data, 4, "Category list has wrong length")

	assert.Equal(suite.T(), c1.Data.Name, categories.Data[0].Name)
	assert.Equal(suite.T(), c2.Data.Name, categories.Data[2].Name)
	assert.Equal(suite.T(), c3.Data.Name, categories.Data[1].Name)
	assert.Equal(suite.T(), c4.Data.Name, categories.Data[3].Name)
}

func (suite *TestSuiteStandard) TestCategoriesPagination() {
	for i := 0; i < 10; i++ {
		createTestCategory(suite.T(), v4.CategoryEditable{Name: fmt.Sprint(i)})
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
			r := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v4/categories?offset=%d&limit=%d", tt.offset, tt.limit), "")
			test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)

			var categories v4.CategoryListResponse
			test.DecodeResponse(t, &r, &categories)

			assert.Equal(suite.T(), tt.offset, categories.Pagination.Offset)
			assert.Equal(suite.T(), tt.limit, categories.Pagination.Limit)
			assert.Equal(suite.T(), tt.expectedCount, categories.Pagination.Count)
			assert.Equal(suite.T(), tt.expectedTotal, categories.Pagination.Total)
		})
	}
}
