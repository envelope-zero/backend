package controllers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func createTestAccount(t *testing.T, c models.AccountCreate) controllers.AccountResponse {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = createTestBudget(t, models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	r := test.Request(t, http.MethodPost, "http://example.com/v1/accounts", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var a controllers.AccountResponse
	test.DecodeResponse(t, &r, &a)

	return a
}

func (suite *TestSuiteEnv) TestGetAccounts() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts", "")

	var response controllers.AccountListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	if !assert.Len(suite.T(), response.Data, 3) {
		assert.FailNow(suite.T(), "Response does not have exactly 3 items")
	}

	bankAccount := response.Data[0]
	assert.Equal(suite.T(), "Bank Account", bankAccount.Name)
	assert.Equal(suite.T(), true, bankAccount.OnBudget)
	assert.Equal(suite.T(), false, bankAccount.External)

	cashAccount := response.Data[1]
	assert.Equal(suite.T(), "Cash Account", cashAccount.Name)
	assert.Equal(suite.T(), false, cashAccount.OnBudget)
	assert.Equal(suite.T(), false, cashAccount.External)

	externalAccount := response.Data[2]
	assert.Equal(suite.T(), "External Account", externalAccount.Name)
	assert.Equal(suite.T(), false, externalAccount.OnBudget)
	assert.Equal(suite.T(), true, externalAccount.External)

	for _, account := range response.Data {
		assert.LessOrEqual(suite.T(), time.Since(account.CreatedAt), test.TOLERANCE)
		assert.LessOrEqual(suite.T(), time.Since(account.UpdatedAt), test.TOLERANCE)
	}

	if !decimal.NewFromFloat(-30).Equal(bankAccount.Balance) {
		assert.Fail(suite.T(), "Account balance does not equal -30", bankAccount.Balance)
	}

	if !decimal.NewFromFloat(-10).Equal(bankAccount.ReconciledBalance) {
		assert.Fail(suite.T(), "Account reconciled balance does not equal -10", bankAccount.ReconciledBalance)
	}

	if !cashAccount.ReconciledBalance.IsZero() {
		assert.Fail(suite.T(), "Account reconciled balance does not equal 0", cashAccount.ReconciledBalance)
	}

	if !decimal.NewFromFloat(10).Equal(externalAccount.ReconciledBalance) {
		assert.Fail(suite.T(), "Account reconciled balance does not equal 10", externalAccount.ReconciledBalance)
	}
}

func (suite *TestSuiteEnv) TestNoAccountNotFound() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts/39633f90-3d9f-4b1e-ac24-c341c432a6e3", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestAccountInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/accounts/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/accounts/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/accounts/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/accounts/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateAccount() {
	_ = createTestAccount(suite.T(), models.AccountCreate{Name: "Test account for creation"})
}

func (suite *TestSuiteEnv) TestCreateAccountNoBudget() {
	r := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/accounts", models.Account{})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateBrokenAccount() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/accounts", `{ "createdAt": "New Account", "note": "More tests for accounts to ensure less brokenness something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateAccountNoBody() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/accounts", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateAccountNonExistingBudget() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/accounts", models.AccountCreate{BudgetID: uuid.New()})
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestGetAccount() {
	a := createTestAccount(suite.T(), models.AccountCreate{})

	r := test.Request(suite.T(), http.MethodGet, a.Data.Links.Self, "")
	assert.Equal(suite.T(), http.StatusOK, r.Code)
}

func (suite *TestSuiteEnv) TestGetAccountTransactionsNonExistingAccount() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts/57372/transactions", "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code)
}

func (suite *TestSuiteEnv) TestUpdateAccount() {
	a := createTestAccount(suite.T(), models.AccountCreate{Name: "Original name"})

	r := test.Request(suite.T(), http.MethodPatch, a.Data.Links.Self, models.AccountCreate{Name: "Updated new account for testing"})
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)

	var u controllers.AccountResponse
	test.DecodeResponse(suite.T(), &r, &u)

	assert.Equal(suite.T(), "Updated new account for testing", u.Data.Name)
}

func (suite *TestSuiteEnv) TestUpdateAccountBroken() {
	a := createTestAccount(suite.T(), models.AccountCreate{
		Name: "New Account",
		Note: "More tests something something",
	})

	r := test.Request(suite.T(), http.MethodPatch, a.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingAccount() {
	recorder := test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/accounts/9b81de41-eead-451d-bc6b-31fceedd236c", models.AccountCreate{Name: "This account does not exist"})
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteAccountsAndEmptyList() {
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts", "")

	var l controllers.AccountListResponse
	test.DecodeResponse(suite.T(), &r, &l)

	for _, a := range l.Data {
		r = test.Request(suite.T(), http.MethodDelete, a.Links.Self, "")
		test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &r)
	}

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts", "")
	test.DecodeResponse(suite.T(), &r, &l)

	// Verify that the account list is an empty list, not null
	assert.NotNil(suite.T(), l.Data)
	assert.Empty(suite.T(), l.Data)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingAccount() {
	recorder := test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/accounts/77b70a75-4bb3-4d1d-90cf-5b7a61f452f5", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteAccountWithBody() {
	a := createTestAccount(suite.T(), models.AccountCreate{Name: "Delete me now!"})

	r := test.Request(suite.T(), http.MethodDelete, a.Data.Links.Self, models.AccountCreate{Name: "Some other account"})
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &r)

	r = test.Request(suite.T(), http.MethodGet, a.Data.Links.Self, "")
}
