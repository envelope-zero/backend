package v4_test

import (
	"fmt"
	"net/http"
	"testing"

	v4 "github.com/envelope-zero/backend/v4/pkg/controllers/v4"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TODO: migrate all createTest* methods to functions with *testing.T as first argument.
func (suite *TestSuiteStandard) createTestMatchRule(t *testing.T, matchRule v4.MatchRuleEditable, expectedStatus ...int) v4.MatchRuleResponse {
	// Default to 201 Creted as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	rules := []v4.MatchRuleEditable{matchRule}

	r := test.Request(t, http.MethodPost, "http://example.com/v4/match-rules", rules)
	test.AssertHTTPStatus(t, &r, expectedStatus...)

	var res v4.MatchRuleCreateResponse
	test.DecodeResponse(t, &r, &res)

	return res.Data[0]
}

func (suite *TestSuiteStandard) TestMatchRuleCreate() {
	a := suite.createTestAccount(suite.T(), v4.AccountEditable{Name: "TestMatchRuleCreate"})

	tests := []struct {
		name           string
		create         []models.MatchRule
		expectedErrors []string
		expectedStatus int
	}{
		{
			"All successful",
			[]models.MatchRule{
				{
					Priority:  10,
					Match:     "Some Match*",
					AccountID: a.Data.ID,
				},
				{
					Priority:  10,
					Match:     "Bank*",
					AccountID: a.Data.ID,
				},
			},
			[]string{
				"",
				"",
			},
			http.StatusCreated,
		},
		{
			"Second fails",
			[]models.MatchRule{
				{
					Priority:  10,
					Match:     "Bank*",
					AccountID: a.Data.ID,
				},
				{
					Priority:  10,
					Match:     "Bank*",
					AccountID: uuid.New(),
				},
			},
			[]string{
				"",
				"there is no Account with this ID",
			},
			http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPost, "http://example.com/v4/match-rules", tt.create)
			test.AssertHTTPStatus(t, &r, tt.expectedStatus)

			var tr v4.MatchRuleCreateResponse
			test.DecodeResponse(t, &r, &tr)

			for i, r := range tr.Data {
				if tt.expectedErrors[i] != "" {
					assert.Equal(t, tt.expectedErrors[i], *r.Error)
				} else {
					assert.Equal(t, fmt.Sprintf("http://example.com/v4/match-rules/%s", r.Data.ID), r.Data.Links.Self)
				}
			}
		})
	}
}

// TestMatchRulesOptions verifies that the HTTP OPTIONS response for //match-rules/{id} is correct.
func (suite *TestSuiteStandard) TestMatchRulesOptions() {
	tests := []struct {
		name     string                    // Name for the test
		status   int                       // Expected HTTP status
		id       string                    // String to use as ID. Ignored when pathFunc is non-nil
		pathFunc func(t *testing.T) string // Function returning the path
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
			func(t *testing.T) string {
				return suite.createTestMatchRule(t, v4.MatchRuleEditable{
					AccountID: suite.createTestAccount(t, v4.AccountEditable{}).Data.ID,
					Match:     "TestMatch*",
				}).Data.Links.Self
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var p string
			if tt.pathFunc != nil {
				p = tt.pathFunc(t)
			} else {
				p = fmt.Sprintf("%s/%s", "http://example.com/v4/match-rules", tt.id)
			}

			r := test.Request(t, http.MethodOptions, p, "")
			test.AssertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestMatchRulesDatabaseError verifies that the endpoints return the appropriate
// error when the database is disconncted.
func (suite *TestSuiteStandard) TestMatchRulesDatabaseError() {
	tests := []struct {
		name   string // Name of the test
		path   string // Path to send request to
		method string // HTTP method to use
		body   string // The request body
	}{
		{"GET Collection", "", http.MethodGet, ""},
		// Skipping POST Collection here since we need to check the indivdual Match Rules for that one
		{"OPTIONS Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodOptions, ""},
		{"GET Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodGet, ""},
		{"PATCH Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodPatch, ""},
		{"DELETE Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodDelete, ""},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			suite.CloseDB()

			recorder := test.Request(t, tt.method, fmt.Sprintf("http://example.com/v4/match-rules%s", tt.path), tt.body)
			test.AssertHTTPStatus(t, &recorder, http.StatusInternalServerError)
			assert.Equal(t, httperrors.ErrDatabaseClosed.Error(), test.DecodeError(t, recorder.Body.Bytes()))
		})
	}
}

// TestMatchRulesGetFilter verifies that filtering Match Rules works as expected.
func (suite *TestSuiteStandard) TestMatchRulesGetFilter() {
	b := suite.createTestBudget(suite.T(), v4.BudgetEditable{})

	a1 := suite.createTestAccount(suite.T(), v4.AccountEditable{BudgetID: b.Data.ID, Name: "TestMatchRulesGetFilter 1"})
	a2 := suite.createTestAccount(suite.T(), v4.AccountEditable{BudgetID: b.Data.ID, Name: "TestMatchRulesGetFilter 2"})

	_ = suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  1,
		Match:     "Testing A Match*",
		AccountID: a1.Data.ID,
	})

	_ = suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  2,
		Match:     "*Match the Second Account",
		AccountID: a2.Data.ID,
	})

	_ = suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  1,
		Match:     "Exact match",
		AccountID: a2.Data.ID,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Limit over count", "limit=5", 3},
		{"Limit under count", "limit=2", 2},
		{"Offset", "offset=2", 1},
		{"Account ID", fmt.Sprintf("account=%s", a2.Data.ID), 2},
		{"Priority", "priority=1", 2},
		{"Non-existent account", fmt.Sprintf("account=%s", uuid.New().String()), 0},
		{"Match with content", "match=match", 3},
		{"Empty match", "match=", 0},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re v4.MatchRuleListResponse
			r := test.Request(t, http.MethodGet, fmt.Sprintf("/v4/match-rules?%s", tt.query), "")
			test.AssertHTTPStatus(t, &r, http.StatusOK)
			test.DecodeResponse(t, &r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

// TestMatchRulesGetFilterErrors verifies that filtering Match Rules returns errors as expected.
func (suite *TestSuiteStandard) TestMatchRulesGetFilterErrors() {
	b := suite.createTestBudget(suite.T(), v4.BudgetEditable{})

	a1 := suite.createTestAccount(suite.T(), v4.AccountEditable{BudgetID: b.Data.ID, Name: "TestMatchRulesGetFilter 1"})
	a2 := suite.createTestAccount(suite.T(), v4.AccountEditable{BudgetID: b.Data.ID, Name: "TestMatchRulesGetFilter 2"})

	_ = suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  1,
		Match:     "Testing A Match*",
		AccountID: a1.Data.ID,
	})

	_ = suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  2,
		Match:     "*Match the Second Account",
		AccountID: a2.Data.ID,
	})

	_ = suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  1,
		Match:     "Exact match",
		AccountID: a2.Data.ID,
	})

	tests := []struct {
		name   string
		query  string
		status int
	}{
		{"Invalid UUID", "account=MorreWroteThis", http.StatusBadRequest},
		{"Invalid query string", "&test&% 20hello", http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re v4.MatchRuleListResponse
			r := test.Request(t, http.MethodGet, fmt.Sprintf("/v4/match-rules?%s", tt.query), "")
			test.AssertHTTPStatus(t, &r, tt.status)
			test.DecodeResponse(t, &r, &re)
		})
	}
}

// TestMatchRulesCreateInvalidBody verifies that creation of Match Rules
// with an unparseable request body returns a HTTP Bad Request.
func (suite *TestSuiteStandard) TestMatchRulesCreateInvalidBody() {
	r := test.Request(suite.T(), http.MethodPost, "http://example.com/v4/match-rules", `{ Invalid request": Body }`)
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	var tr v4.MatchRuleCreateResponse
	test.DecodeResponse(suite.T(), &r, &tr)

	assert.Equal(suite.T(), httperrors.ErrInvalidBody.Error(), *tr.Error)
	assert.Nil(suite.T(), tr.Data)
}

// TestMatchRulesCreate verifies that transaction creation works.
func (suite *TestSuiteStandard) TestMatchRulesCreate() {
	budget := suite.createTestBudget(suite.T(), v4.BudgetEditable{})
	internalAccount := suite.createTestAccount(suite.T(), v4.AccountEditable{External: false, BudgetID: budget.Data.ID, Name: "TestMatchRulesCreate Internal"})

	tests := []struct {
		name           string
		matchRules     []models.MatchRule
		expectedStatus int
		expectedError  *error   // Error expected in the response
		expectedErrors []string // Errors expected for the individual transactions
	}{
		{
			"One success, one fail",
			[]models.MatchRule{
				{
					AccountID: internalAccount.Data.ID,
				},
				{
					AccountID: uuid.New(),
				},
			},
			http.StatusNotFound,
			nil,
			[]string{
				"",
				"there is no Account with this ID",
			},
		},
		{
			"Two success",
			[]models.MatchRule{
				{
					AccountID: internalAccount.Data.ID,
					Match:     "* glob glob glob *",
				},
				{
					AccountID: internalAccount.Data.ID,
					Match:     "Test Match 2",
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
			r := test.Request(t, http.MethodPost, "http://example.com/v4/match-rules", tt.matchRules)
			test.AssertHTTPStatus(t, &r, tt.expectedStatus)

			var tr v4.MatchRuleCreateResponse
			test.DecodeResponse(t, &r, &tr)

			for i, mr := range tr.Data {
				if tt.expectedErrors[i] == "" {
					assert.Equal(t, fmt.Sprintf("http://example.com/v4/match-rules/%s", mr.Data.ID), mr.Data.Links.Self)
				} else {
					// This needs to be in the else to prevent nil pointer errors since we're dereferencing pointers
					assert.Equal(t, tt.expectedErrors[i], *mr.Error)
				}
			}
		})
	}
}

// TestMatchRulesGetSingle verifies that a Match Rule can be read from the API via its link
// and that the link is for API v4.
func (suite *TestSuiteStandard) TestMatchRulesGetSingle() {
	tests := []struct {
		name     string                    // Name for the test
		status   int                       // Expected HTTP status
		id       string                    // String to use as ID. Ignored when pathFunc is non-nil
		pathFunc func(t *testing.T) string // Function returning the path
	}{
		{
			"Standard transaction",
			http.StatusOK,
			"",
			func(t *testing.T) string {
				return suite.createTestMatchRule(t, v4.MatchRuleEditable{AccountID: suite.createTestAccount(t, v4.AccountEditable{}).Data.ID}).Data.Links.Self
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
				p = tt.pathFunc(t)
			} else {
				p = fmt.Sprintf("%s/%s", "http://example.com/v4/match-rules", tt.id)
			}

			r := test.Request(suite.T(), http.MethodGet, p, "")
			test.AssertHTTPStatus(suite.T(), &r, tt.status)
		})
	}
}

// TestMatchRulesUpdateFail verifies that transaction updates fail where they should.
func (suite *TestSuiteStandard) TestMatchRulesUpdateFail() {
	m := suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		AccountID: suite.createTestAccount(suite.T(), v4.AccountEditable{}).Data.ID,
		Match:     "Some match*",
	})

	tests := []struct {
		name   string // Name for the test
		status int    // Expected HTTP status
		body   any    // Body to send to the PATCH endpoint
		path   string // Path to send the PATCH request to
	}{
		{
			"Invalid body",
			http.StatusBadRequest,
			`{ "priority": 2" }`,
			m.Data.Links.Self,
		},
		{
			"Invalid type",
			http.StatusBadRequest,
			map[string]any{
				"match": false,
			},
			m.Data.Links.Self,
		},
		{
			"Non-existing account",
			http.StatusNotFound,
			`{ "accountId": "e6fa8eb5-5f2c-4292-8ef9-02f0c2af1ce4" }`,
			m.Data.Links.Self,
		},
		{
			"Invalid path",
			http.StatusBadRequest,
			"",
			"http://example.com/v4/match-rules/NotAUUID",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPatch, tt.path, tt.body)
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestMatchRulesUpdate verifies that transaction updates are successful.
func (suite *TestSuiteStandard) TestMatchRulesUpdate() {
	m := suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		AccountID: suite.createTestAccount(suite.T(), v4.AccountEditable{}).Data.ID,
		Match:     "Some match*",
	})

	newAccount := suite.createTestAccount(suite.T(), v4.AccountEditable{})

	tests := []struct {
		name string // Name for the test
		body any    // Body to send to the PATCH endpoint
	}{
		{
			"Change match",
			map[string]string{
				"match": "Some match more exactly*",
			},
		},
		{
			"Change priority and match",
			map[string]any{
				"priority": 1487,
				"match":    "return 4;",
			},
		},
		{
			"Change account",
			map[string]any{
				"accountId": newAccount.Data.ID,
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPatch, m.Data.Links.Self, tt.body)
			test.AssertHTTPStatus(t, &r, http.StatusOK)
		})
	}
}

// TestMatchRulesDelete verifies the correct success and error responses
// for DELETE requests.
func (suite *TestSuiteStandard) TestMatchRulesDelete() {
	tests := []struct {
		name   string // Name for the test
		status int    // Expected HTTP status
		id     string // String to use as ID.
	}{
		{
			"Standard deletion",
			http.StatusNoContent,
			suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
				AccountID: suite.createTestAccount(suite.T(), v4.AccountEditable{}).Data.ID,
				Match:     "Some match*",
			}).Data.ID.String(),
		},
		{
			"Does not exist",
			http.StatusNotFound,
			"4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501",
		},
		{
			"Null UUID",
			http.StatusBadRequest,
			"00000000-0000-0000-0000-000000000000",
		},
		{
			"Invalid UUID",
			http.StatusBadRequest,
			"NotAUUID",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s", "http://example.com/v4/match-rules", tt.id)

			r := test.Request(t, http.MethodDelete, p, "")
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestMatchRulesGetSorted verifies that Match Rules are sorted as expected.
func (suite *TestSuiteStandard) TestMatchRulesGetSorted() {
	b := suite.createTestBudget(suite.T(), v4.BudgetEditable{})
	a := suite.createTestAccount(suite.T(), v4.AccountEditable{BudgetID: b.Data.ID, Name: "TestMatchRulesGetFilter 1"})

	m1 := suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  1,
		Match:     "Testing A Match*",
		AccountID: a.Data.ID,
	})

	m2 := suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  2,
		Match:     "*Match the Second Account",
		AccountID: a.Data.ID,
	})

	m3 := suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  1,
		Match:     "Exact match",
		AccountID: a.Data.ID,
	})

	m4 := suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  3,
		Match:     "Coffee Shop*",
		AccountID: a.Data.ID,
	})

	m5 := suite.createTestMatchRule(suite.T(), v4.MatchRuleEditable{
		Priority:  3,
		Match:     "Coffee Shop",
		AccountID: a.Data.ID,
	})

	var re v4.MatchRuleListResponse
	r := test.Request(suite.T(), http.MethodGet, "/v4/match-rules", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
	test.DecodeResponse(suite.T(), &r, &re)

	// Lowest priority, alphabetically first
	assert.Equal(suite.T(), *m3.Data, re.Data[0])

	// Lowest priority, alphabetically second
	assert.Equal(suite.T(), *m1.Data, re.Data[1])

	// Higher priority
	assert.Equal(suite.T(), *m2.Data, re.Data[2])

	// Highest priority, alphabetically first
	assert.Equal(suite.T(), *m5.Data, re.Data[3])

	// Highest priority, alphabetically second
	assert.Equal(suite.T(), *m4.Data, re.Data[4])
}
