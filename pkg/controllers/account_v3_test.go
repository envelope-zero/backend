package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestAccountV3(t *testing.T, c models.AccountCreate, expectedStatus ...int) controllers.AccountResponseV3 {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = suite.createTestBudgetV3(t, models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	body := []models.AccountCreate{
		c,
	}

	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/accounts", body)
	assertHTTPStatus(t, &r, expectedStatus...)

	var a controllers.AccountCreateResponseV3
	suite.decodeResponse(&r, &a)

	if r.Code == http.StatusCreated {
		return a.Data[0]
	}

	return controllers.AccountResponseV3{}
}

// TestAccountsV3DBClosed verifies that errors are processed correctly when
// the database is closed.
func (suite *TestSuiteStandard) TestAccountsV3DBClosed() {
	b := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	tests := []struct {
		name string             // Name of the test
		test func(t *testing.T) // Code to run
	}{
		{
			"Creation fails",
			func(t *testing.T) {
				suite.createTestAccountV3(t, models.AccountCreate{BudgetID: b.Data.ID}, http.StatusInternalServerError)
			},
		},
		{
			"GET fails",
			func(t *testing.T) {
				recorder := test.Request(suite.controller, t, http.MethodGet, "http://example.com/v3/accounts", "")
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

// TestAccountV3Options verifies that OPTIONS requests are handled correctly.
func (suite *TestSuiteStandard) TestAccountsV3Options() {
	tests := []struct {
		name   string
		id     string // path at the Accounts endpoint to test
		status int    // Expected HTTP status code
	}{
		{"No account with this ID", uuid.New().String(), http.StatusNotFound},
		{"Not a valid UUID", "NotParseableAsUUID", http.StatusBadRequest},
		{"Account exists", suite.createTestAccountV3(suite.T(), models.AccountCreate{}).Data.ID.String(), http.StatusNoContent},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", "http://example.com/v3/accounts", tt.id)
			r := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
			assertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestAccountV3GetSingle verifies that requests for the resource endpoints are
// handled correctly.
func (suite *TestSuiteStandard) TestAccountsV3GetSingle() {
	a := suite.createTestAccountV3(suite.T(), models.AccountCreate{})

	tests := []struct {
		name   string
		id     string
		status int
		method string
	}{
		{"GET Existing account", a.Data.ID.String(), http.StatusOK, http.MethodGet},
		{"GET ID nil", uuid.Nil.String(), http.StatusBadRequest, http.MethodGet},
		{"GET No account with this ID", uuid.New().String(), http.StatusNotFound, http.MethodGet},
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
			r := test.Request(suite.controller, t, tt.method, fmt.Sprintf("http://example.com/v3/accounts/%s", tt.id), "")

			var budget controllers.AccountResponseV3
			suite.decodeResponse(&r, &budget)
			assertHTTPStatus(t, &r, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestAccountsV3GetFilter() {
	b1 := suite.createTestBudget(models.BudgetCreate{})
	b2 := suite.createTestBudget(models.BudgetCreate{})

	_ = suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name:     "Exact Account Match",
		Note:     "This is a specific note",
		BudgetID: b1.Data.ID,
		OnBudget: true,
		External: false,
	})

	_ = suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name:     "External Account Filter",
		Note:     "This is a specific note",
		BudgetID: b2.Data.ID,
		OnBudget: true,
		External: true,
	})

	_ = suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name:     "External Account Filter",
		Note:     "A different note",
		BudgetID: b1.Data.ID,
		OnBudget: false,
		External: true,
		Hidden:   true,
	})

	_ = suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name:     "",
		Note:     "specific note",
		BudgetID: b1.Data.ID,
	})

	_ = suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name:     "Name only",
		Note:     "",
		BudgetID: b1.Data.ID,
		Hidden:   true,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Name single", "name=Exact Account Match", 1},
		{"Name multiple", "name=External Account Filter", 2},
		{"Fuzzy name", "name=Account", 3},
		{"Note", "note=A different note", 1},
		{"Fuzzy Note", "note=note", 4},
		{"Empty name with note", "name=&note=specific", 1},
		{"Empty note with name", "note=&name=Name", 1},
		{"Empty note and name", "note=&name=&onBudget=false", 0},
		{"Budget", fmt.Sprintf("budget=%s", b1.Data.ID), 4},
		{"On budget", "onBudget=true", 1},
		{"Off budget", "onBudget=false", 4},
		{"External", "external=true", 2},
		{"Internal", "external=false", 3},
		{"Not Hidden", "hidden=false", 3},
		{"Hidden", "hidden=true", 2},
		{"Search for 'na", "search=na", 3},
		{"Search for 'fi", "search=fi", 4},
		{"Offset 2", "offset=2", 3},
		{"Offset 2, limit 2", "offset=2&limit=2", 2},
		{"Limit 4", "limit=4", 4},
		{"Limit 0", "limit=0", 0},
		{"Limit -1", "limit=-1", 5},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.BudgetListResponse
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v3/accounts?%s", tt.query), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			var accountNames []string
			for _, d := range re.Data {
				accountNames = append(accountNames, d.Name)
			}
			assert.Equal(t, tt.len, len(re.Data), "Existing accounts: %#v, Request-ID: %s", strings.Join(accountNames, ", "), r.Header().Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestAccountsV3CreateFails() {
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
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			recorder := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/accounts", tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestAccountsV3Update() {
	a := suite.createTestAccountV3(suite.T(), models.AccountCreate{Name: "Original name", OnBudget: true})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"name":     "Updated new account for testing",
		"note":     "",
		"onBudget": false,
	})
	assertHTTPStatus(suite.T(), &r, http.StatusOK)

	var u controllers.AccountResponse
	suite.decodeResponse(&r, &u)

	assert.Equal(suite.T(), "Updated new account for testing", u.Data.Name)
	assert.Equal(suite.T(), "", u.Data.Note)
	assert.Equal(suite.T(), false, u.Data.OnBudget)
}

func (suite *TestSuiteStandard) TestAccountsV3UpdateFails() {
	tests := []struct {
		name   string
		id     string
		body   any
		status int // expected response status
	}{
		{"Invalid type", "", `{"name": 2}`, http.StatusBadRequest},
		{"Broken JSON", "", `{ "name": 2" }`, http.StatusBadRequest},
		{"Non-existing account", uuid.New().String(), `{"name": 2}`, http.StatusNotFound},
		{"Set budget to uuid.Nil", "", models.AccountCreate{}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				// Create test Account
				budget := suite.createTestAccountV3(suite.T(), models.AccountCreate{
					Name: "New Budget",
					Note: "More tests something something",
				})

				tt.id = budget.Data.ID.String()
			}

			// Update Account
			recorder = test.Request(suite.controller, t, http.MethodPatch, fmt.Sprintf("http://example.com/v3/accounts/%s", tt.id), tt.body)
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestAccountsV3Delete verifies all cases for Account deletions.
func (suite *TestSuiteStandard) TestAccountsV3Delete() {
	tests := []struct {
		name   string
		id     string
		status int // expected response status
	}{
		{"Success", "", http.StatusNoContent},
		{"Non-existing Account", uuid.New().String(), http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				// Create test Account
				a := suite.createTestAccountV3(t, models.AccountCreate{})
				tt.id = a.Data.ID.String()
			}

			// Delete Account
			recorder = test.Request(suite.controller, t, http.MethodDelete, fmt.Sprintf("http://example.com/v3/accounts/%s", tt.id), "")
			assertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestAccountsV3GetSorted verifies that Accounts are sorted by name.
func (suite *TestSuiteStandard) TestAccountsV3GetSorted() {
	a1 := suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name: "Alphabetically first",
	})

	a2 := suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name: "Second in creation, third in list",
	})

	a3 := suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name: "First is alphabetically second",
	})

	a4 := suite.createTestAccountV3(suite.T(), models.AccountCreate{
		Name: "Zulu is the last one",
	})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v3/accounts", "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)

	var accounts controllers.AccountListResponseV3
	suite.decodeResponse(&r, &accounts)

	if !assert.Len(suite.T(), accounts.Data, 4) {
		assert.FailNow(suite.T(), "Account list has wrong length")
	}

	assert.Equal(suite.T(), a1.Data.Name, accounts.Data[0].Name)
	assert.Equal(suite.T(), a2.Data.Name, accounts.Data[2].Name)
	assert.Equal(suite.T(), a3.Data.Name, accounts.Data[1].Name)
	assert.Equal(suite.T(), a4.Data.Name, accounts.Data[3].Name)
}

func (suite *TestSuiteStandard) TestAccountsV3Pagination() {
	for i := 0; i < 10; i++ {
		suite.createTestAccountV3(suite.T(), models.AccountCreate{Name: fmt.Sprint(i)})
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
			r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v3/accounts?offset=%d&limit=%d", tt.offset, tt.limit), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)

			var accounts controllers.AccountListResponseV3
			suite.decodeResponse(&r, &accounts)

			assert.Equal(suite.T(), tt.offset, accounts.Pagination.Offset)
			assert.Equal(suite.T(), tt.limit, accounts.Pagination.Limit)
			assert.Equal(suite.T(), tt.expectedCount, accounts.Pagination.Count)
			assert.Equal(suite.T(), tt.expectedTotal, accounts.Pagination.Total)
		})
	}
}
