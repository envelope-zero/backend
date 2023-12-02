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

func (suite *TestSuiteStandard) createTestCategoryV3(t *testing.T, c models.CategoryCreate, expectedStatus ...int) controllers.CategoryResponseV3 {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = suite.createTestBudget(models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	if c.Name == "" {
		c.Name = uuid.NewString()
	}

	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	body := []models.CategoryCreate{c}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/categories", body)
	assertHTTPStatus(t, &r, expectedStatus...)

	var category controllers.CategoryCreateResponseV3
	suite.decodeResponse(&r, &category)

	if r.Code == http.StatusCreated {
		return category.Data[0]
	}

	return controllers.CategoryResponseV3{}
}

// TestCategoriesV3DBClosed verifies that errors are processed correctly when
// the database is closed.
func (suite *TestSuiteStandard) TestCategoriesV3DBClosed() {
	b := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	tests := []struct {
		name string             // Name of the test
		test func(t *testing.T) // Code to run
	}{
		{
			"Creation fails",
			func(t *testing.T) {
				suite.createTestCategoryV3(t, models.CategoryCreate{BudgetID: b.Data.ID}, http.StatusInternalServerError)
			},
		},
		{
			"GET fails",
			func(t *testing.T) {
				recorder := test.Request(suite.controller, t, http.MethodGet, "http://example.com/v3/categories", "")
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

// TestCategoriesV3Options verifies that OPTIONS requests are handled correctly.
func (suite *TestSuiteStandard) TestCategoriesV3Options() {
	tests := []struct {
		name   string
		id     string // path at the Accounts endpoint to test
		status int    // Expected HTTP status code
	}{
		{"No Category with this ID", uuid.New().String(), http.StatusNotFound},
		{"Not a valid UUID", "NotParseableAsUUID", http.StatusBadRequest},
		{"Category exists", suite.createTestCategoryV3(suite.T(), models.CategoryCreate{}).Data.ID.String(), http.StatusNoContent},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", "http://example.com/v3/categories", tt.id)
			r := test.Request(suite.controller, t, http.MethodOptions, path, "")
			assertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestCategoriesV3GetSingle verifies that requests for the resource endpoints are
// handled correctly.
func (suite *TestSuiteStandard) TestCategoriesV3GetSingle() {
	c := suite.createTestCategoryV3(suite.T(), models.CategoryCreate{})

	tests := []struct {
		name   string
		id     string
		status int
		method string
	}{
		{"GET Existing Category", c.Data.ID.String(), http.StatusOK, http.MethodGet},
		{"GET ID nil", uuid.Nil.String(), http.StatusBadRequest, http.MethodGet},
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
			r := test.Request(suite.controller, t, tt.method, fmt.Sprintf("http://example.com/v3/categories/%s", tt.id), "")

			var category controllers.CategoryResponseV3
			suite.decodeResponse(&r, &category)
			assertHTTPStatus(t, &r, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestCategoriesV3GetFilter() {
	b1 := suite.createTestBudget(models.BudgetCreate{})
	b2 := suite.createTestBudget(models.BudgetCreate{})

	_ = suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
		Name:     "Category Name",
		Note:     "A note for this category",
		BudgetID: b1.Data.ID,
		Hidden:   true,
	})

	_ = suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
		Name:     "Groceries",
		Note:     "For Groceries",
		BudgetID: b2.Data.ID,
	})

	_ = suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
		Name:     "Daily stuff",
		Note:     "Groceries, Drug Store, …",
		BudgetID: b2.Data.ID,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Budget 1", fmt.Sprintf("budget=%s", b1.Data.ID), 1},
		{"Budget Not Existing", "budget=c9e4ee7a-e702-4f92-b168-11a95b22c7aa", 0},
		{"Empty Note", "note=", 0},
		{"Empty Name", "name=", 0},
		{"Name & Note", "name=Category Name&note=A note for this category", 1},
		{"Fuzzy name, no note", "name=Category&note=", 0},
		{"Fuzzy name", "name=t", 2},
		{"Fuzzy note, no name", "name=&note=Groceries", 0},
		{"Fuzzy note", "note=Groceries", 2},
		{"Not archived", "archived=false", 2},
		{"Archived", "archived=true", 1},
		{"Search for 'groceries'", "search=groceries", 2},
		{"Search for 'FOR'", "search=FOR", 2},
		{"Offset 2", "offset=2", 1},
		{"Offset 0, limit 2", "offset=0&limit=2", 2},
		{"Limit 4", "limit=4", 3},
		{"Limit 0", "limit=0", 0},
		{"Limit -1", "limit=-1", 3},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.CategoryListResponse
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v3/categories?%s", tt.query), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestCategoriesV3CreateFails() {
	// Test category for uniqueness
	c := suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
		Name: "Unique Category Name for Budget",
	})

	tests := []struct {
		name   string
		body   any
		status int // expected HTTP status
	}{
		{"Broken Body", `[{ "note": 2 }]`, http.StatusBadRequest},
		{"No body", "", http.StatusBadRequest},
		{
			"No Budget",
			`[{ "note": "Some text" }]`,
			http.StatusBadRequest,
		},
		{
			"Non-existing Budget",
			`[{ "budgetId": "ea85ad1a-3679-4ced-b83b-89566c12ece9" }]`,
			http.StatusBadRequest,
		},
		{
			"Duplicate name in Budget",
			models.CategoryCreate{
				BudgetID: c.Data.BudgetID,
				Name:     c.Data.Name,
			},
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			recorder := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/categories", tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestCategoriesV3Update() {
	envelope := suite.createTestCategoryV3(suite.T(), models.CategoryCreate{Name: "New Category", Note: "Keks is a cuddly cat"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, envelope.Data.Links.Self, map[string]any{
		"name": "Updated new Category for testing",
		"note": "",
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedCategory controllers.CategoryResponseV3
	suite.decodeResponse(&recorder, &updatedCategory)

	assert.Equal(suite.T(), "", updatedCategory.Data.Note)
	assert.Equal(suite.T(), "Updated new Category for testing", updatedCategory.Data.Name)
}

func (suite *TestSuiteStandard) TestCategoriesV3UpdateFails() {
	tests := []struct {
		name   string
		id     string
		body   any
		status int // expected response status
	}{
		{"Invalid type", "", `{"name": 2}`, http.StatusBadRequest},
		{"Broken JSON", "", `{ "name": 2" }`, http.StatusBadRequest},
		{"Non-existing Category", uuid.New().String(), `{"name": 2}`, http.StatusNotFound},
		{"Set Budget to uuid.Nil", "", models.CategoryCreate{}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				envelope := suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
					Name: "New Envelope",
					Note: "Auto-created for test",
				})

				tt.id = envelope.Data.ID.String()
			}

			recorder = test.Request(suite.controller, t, http.MethodPatch, fmt.Sprintf("http://example.com/v3/categories/%s", tt.id), tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestCategoriesV3Delete verifies all cases for Account deletions.
func (suite *TestSuiteStandard) TestCategoriesV3Delete() {
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
				e := suite.createTestCategoryV3(t, models.CategoryCreate{})
				tt.id = e.Data.ID.String()
			}

			// Delete Account
			recorder = test.Request(suite.controller, t, http.MethodDelete, fmt.Sprintf("http://example.com/v3/categories/%s", tt.id), "")
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestCategoriesV3GetSorted verifies that Accounts are sorted by name.
func (suite *TestSuiteStandard) TestCategoriesV3GetSorted() {
	c1 := suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
		Name: "Alphabetically first",
	})

	c2 := suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
		Name: "Second in creation, third in list",
	})

	c3 := suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
		Name: "First is alphabetically second",
	})

	c4 := suite.createTestCategoryV3(suite.T(), models.CategoryCreate{
		Name: "Zulu is the last one",
	})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v3/categories", "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)

	var categories controllers.CategoryListResponseV3
	suite.decodeResponse(&r, &categories)

	if !assert.Len(suite.T(), categories.Data, 4) {
		assert.FailNow(suite.T(), "Category list has wrong length")
	}

	assert.Equal(suite.T(), c1.Data.Name, categories.Data[0].Name)
	assert.Equal(suite.T(), c2.Data.Name, categories.Data[2].Name)
	assert.Equal(suite.T(), c3.Data.Name, categories.Data[1].Name)
	assert.Equal(suite.T(), c4.Data.Name, categories.Data[3].Name)
}

func (suite *TestSuiteStandard) TestCategoriesV3Pagination() {
	for i := 0; i < 10; i++ {
		suite.createTestCategoryV3(suite.T(), models.CategoryCreate{Name: fmt.Sprint(i)})
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
			r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v3/categories?offset=%d&limit=%d", tt.offset, tt.limit), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)

			var categories controllers.CategoryListResponseV3
			suite.decodeResponse(&r, &categories)

			assert.Equal(suite.T(), tt.offset, categories.Pagination.Offset)
			assert.Equal(suite.T(), tt.limit, categories.Pagination.Limit)
			assert.Equal(suite.T(), tt.expectedCount, categories.Pagination.Count)
			assert.Equal(suite.T(), tt.expectedTotal, categories.Pagination.Total)
		})
	}
}
