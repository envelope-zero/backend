package controllers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/envelope-zero/backend/v2/pkg/controllers"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/envelope-zero/backend/v2/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestAccount(c models.AccountCreate, expectedStatus ...int) controllers.AccountResponse {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = suite.createTestBudget(models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/accounts", c)
	suite.assertHTTPStatus(&r, expectedStatus...)

	var a controllers.AccountResponse
	suite.decodeResponse(&r, &a)

	return a
}

func (suite *TestSuiteStandard) TestAccounts() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts", "")
	suite.assertHTTPStatus(&recorder, http.StatusInternalServerError)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteStandard) TestOptionsAccount() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v1/accounts", uuid.New())
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/accounts/NotParseableAsUUID", "")
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = suite.createTestAccount(models.AccountCreate{}).Data.Links.Self
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestGetAccounts() {
	_ = suite.createTestAccount(models.AccountCreate{})

	var response controllers.AccountListResponse
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts", "")
	suite.assertHTTPStatus(&recorder, http.StatusOK)
	suite.decodeResponse(&recorder, &response)

	assert.Len(suite.T(), response.Data, 1)
}

func (suite *TestSuiteStandard) TestGetAccountsInvalidQuery() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts?onBudget=NotABoolean", "")
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)

	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts?budget=8593-not-a-uuid", "")
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestGetAccountsFilter() {
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
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.BudgetListResponse
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v1/accounts?%s", tt.query), "")
			suite.assertHTTPStatus(&r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			var accountNames []string
			for _, d := range re.Data {
				accountNames = append(accountNames, d.Name)
			}
			assert.Equal(t, tt.len, len(re.Data), "Existing accounts: %#v, Request-ID: %s", strings.Join(accountNames, ", "), r.Header().Get("x-request-id"))
		})
	}

	for _, r := range []controllers.BudgetResponse{b1, b2} {
		test.Request(suite.controller, suite.T(), http.MethodDelete, r.Data.Links.Self, "")
	}

	for _, r := range []controllers.AccountResponse{a1, a2, a3} {
		test.Request(suite.controller, suite.T(), http.MethodDelete, r.Data.Links.Self, "")
	}
}

func (suite *TestSuiteStandard) TestNoAccountNotFound() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts/39633f90-3d9f-4b1e-ac24-c341c432a6e3", "")
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestAccountInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts/-56", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts/notANumber", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts/23", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	/*
	 * PATCH
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/accounts/-274", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/accounts/stringRandom", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	/*
	 * DELETE
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/accounts/-274", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/accounts/stringRandom", "")
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateAccount() {
	_ = suite.createTestAccount(models.AccountCreate{Name: "Test account for creation"})
}

func (suite *TestSuiteStandard) TestCreateAccountNoBudget() {
	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/accounts", models.Account{})
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateBrokenAccount() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/accounts", `{ "createdAt": "New Account", "note": "More tests for accounts to ensure less brokenness something" }`)
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateAccountNoBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/accounts", "")
	suite.assertHTTPStatus(&recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateAccountNonExistingBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/accounts", models.AccountCreate{BudgetID: uuid.New()})
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestGetAccount() {
	a := suite.createTestAccount(models.AccountCreate{})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, a.Data.Links.Self, "")
	assert.Equal(suite.T(), http.StatusOK, r.Code)
}

func (suite *TestSuiteStandard) TestGetAccountTransactionsNonExistingAccount() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts/57372/transactions", "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code)
}

func (suite *TestSuiteStandard) TestUpdateAccount() {
	a := suite.createTestAccount(models.AccountCreate{Name: "Original name", OnBudget: true})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"name":     "Updated new account for testing",
		"note":     "",
		"onBudget": false,
	})
	suite.assertHTTPStatus(&r, http.StatusOK)

	var u controllers.AccountResponse
	suite.decodeResponse(&r, &u)

	assert.Equal(suite.T(), "Updated new account for testing", u.Data.Name)
	assert.Equal(suite.T(), "", u.Data.Note)
	assert.Equal(suite.T(), false, u.Data.OnBudget)
}

func (suite *TestSuiteStandard) TestUpdateAccountBrokenJSON() {
	a := suite.createTestAccount(models.AccountCreate{
		Name: "New Account",
		Note: "More tests something something",
	})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, `{ "name": 2" }`)
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateAccountInvalidType() {
	a := suite.createTestAccount(models.AccountCreate{
		Name: "New Account",
		Note: "More tests something something",
	})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, map[string]any{
		"name": 2,
	})
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateAccountInvalidBudgetID() {
	a := suite.createTestAccount(models.AccountCreate{
		Name: "New Account",
		Note: "More tests something something",
	})

	// Sets the BudgetID to uuid.Nil
	r := test.Request(suite.controller, suite.T(), http.MethodPatch, a.Data.Links.Self, models.AccountCreate{})
	suite.assertHTTPStatus(&r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateNonExistingAccount() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/accounts/9b81de41-eead-451d-bc6b-31fceedd236c", models.AccountCreate{Name: "This account does not exist"})
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteAccountsAndEmptyList() {
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts", "")

	var l controllers.AccountListResponse
	suite.decodeResponse(&r, &l)

	for _, a := range l.Data {
		r = test.Request(suite.controller, suite.T(), http.MethodDelete, a.Links.Self, "")
		suite.assertHTTPStatus(&r, http.StatusNoContent)
	}

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/accounts", "")
	suite.decodeResponse(&r, &l)

	// Verify that the account list is an empty list, not null
	assert.NotNil(suite.T(), l.Data)
	assert.Empty(suite.T(), l.Data)
}

func (suite *TestSuiteStandard) TestDeleteNonExistingAccount() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/accounts/77b70a75-4bb3-4d1d-90cf-5b7a61f452f5", "")
	suite.assertHTTPStatus(&recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteAccountWithBody() {
	a := suite.createTestAccount(models.AccountCreate{Name: "Delete me now!"})

	r := test.Request(suite.controller, suite.T(), http.MethodDelete, a.Data.Links.Self, models.AccountCreate{Name: "Some other account"})
	suite.assertHTTPStatus(&r, http.StatusNoContent)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, a.Data.Links.Self, "")
}
