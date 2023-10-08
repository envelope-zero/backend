package controllers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestAccountsV2() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v2/accounts", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusInternalServerError)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteStandard) TestGetAccountsV2() {
	_ = suite.createTestAccount(models.AccountCreate{Name: "TestGetAccounts"})

	var response []models.AccountV2
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v2/accounts", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	suite.decodeResponse(&recorder, &response)

	assert.Len(suite.T(), response, 1)
}

func (suite *TestSuiteStandard) TestGetAccountsV2InvalidQuery() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v2/accounts?onBudget=NotABoolean", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v2/accounts?budget=8593-not-a-uuid", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestGetAccountsV2Filter() {
	b1 := suite.createTestBudget(models.BudgetCreate{})
	b2 := suite.createTestBudget(models.BudgetCreate{})

	a1 := suite.createTestAccount(models.AccountCreate{
		Name:     "Exact Account Match",
		Note:     "This is a specific note",
		BudgetID: b1.Data.ID,
		OnBudget: true,
		External: false,
	})

	a2 := suite.createTestAccount(models.AccountCreate{
		Name:     "External Account Filter",
		Note:     "This is a specific note",
		BudgetID: b2.Data.ID,
		OnBudget: true,
		External: true,
	})

	a3 := suite.createTestAccount(models.AccountCreate{
		Name:     "External Account Filter",
		Note:     "A different note",
		BudgetID: b1.Data.ID,
		OnBudget: false,
		External: true,
		Hidden:   true,
	})

	_ = suite.createTestAccount(models.AccountCreate{
		Name:     "",
		Note:     "specific note",
		BudgetID: b1.Data.ID,
	})

	_ = suite.createTestAccount(models.AccountCreate{
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
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re []models.AccountV2
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v2/accounts?%s", tt.query), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			var accountNames []string
			for _, d := range re {
				accountNames = append(accountNames, d.Name)
			}
			assert.Equal(t, tt.len, len(re), "Existing accounts: %#v, Request-ID: %s", strings.Join(accountNames, ", "), r.Header().Get("x-request-id"))
		})
	}

	for _, r := range []controllers.BudgetResponse{b1, b2} {
		test.Request(suite.controller, suite.T(), http.MethodDelete, r.Data.Links.Self, "")
	}

	for _, r := range []controllers.AccountResponse{a1, a2, a3} {
		test.Request(suite.controller, suite.T(), http.MethodDelete, r.Data.Links.Self, "")
	}
}
