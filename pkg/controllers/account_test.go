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
	r := test.Request(t, http.MethodPost, "http://example.com/v1/accounts", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var a controllers.AccountResponse
	test.DecodeResponse(t, &r, &a)

	return a
}

func TestGetAccounts(t *testing.T) {
	recorder := test.Request(t, http.MethodGet, "http://example.com/v1/accounts", "")

	var response controllers.AccountListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 3) {
		assert.FailNow(t, "Response does not have exactly 3 items")
	}

	bankAccount := response.Data[0]
	assert.Equal(t, "Bank Account", bankAccount.Name)
	assert.Equal(t, true, bankAccount.OnBudget)
	assert.Equal(t, false, bankAccount.External)

	cashAccount := response.Data[1]
	assert.Equal(t, "Cash Account", cashAccount.Name)
	assert.Equal(t, false, cashAccount.OnBudget)
	assert.Equal(t, false, cashAccount.External)

	externalAccount := response.Data[2]
	assert.Equal(t, "External Account", externalAccount.Name)
	assert.Equal(t, false, externalAccount.OnBudget)
	assert.Equal(t, true, externalAccount.External)

	for _, account := range response.Data {
		assert.LessOrEqual(t, time.Since(account.CreatedAt), test.TOLERANCE)
		assert.LessOrEqual(t, time.Since(account.UpdatedAt), test.TOLERANCE)
	}

	if !decimal.NewFromFloat(-30).Equal(bankAccount.Balance) {
		assert.Fail(t, "Account balance does not equal -30", bankAccount.Balance)
	}

	if !decimal.NewFromFloat(-10).Equal(bankAccount.ReconciledBalance) {
		assert.Fail(t, "Account reconciled balance does not equal -10", bankAccount.ReconciledBalance)
	}

	if !cashAccount.ReconciledBalance.IsZero() {
		assert.Fail(t, "Account reconciled balance does not equal 0", cashAccount.ReconciledBalance)
	}

	if !decimal.NewFromFloat(10).Equal(externalAccount.ReconciledBalance) {
		assert.Fail(t, "Account reconciled balance does not equal 10", externalAccount.ReconciledBalance)
	}
}

func TestNoAccountNotFound(t *testing.T) {
	recorder := test.Request(t, http.MethodGet, "http://example.com/v1/accounts/39633f90-3d9f-4b1e-ac24-c341c432a6e3", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestAccountInvalidIDs(t *testing.T) {
	/*
	 *  GET
	 */
	r := test.Request(t, http.MethodGet, "http://example.com/v1/accounts/-56", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "http://example.com/v1/accounts/notANumber", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "http://example.com/v1/accounts/23", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(t, http.MethodPatch, "http://example.com/v1/accounts/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodPatch, "http://example.com/v1/accounts/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(t, http.MethodDelete, "http://example.com/v1/accounts/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodDelete, "http://example.com/v1/accounts/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateAccount(t *testing.T) {
	_ = createTestAccount(t, models.AccountCreate{Name: "Test account for creation"})
}

func TestCreateBrokenAccount(t *testing.T) {
	recorder := test.Request(t, http.MethodPost, "http://example.com/v1/accounts", `{ "createdAt": "New Account", "note": "More tests for accounts to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAccountNoBody(t *testing.T) {
	recorder := test.Request(t, http.MethodPost, "http://example.com/v1/accounts", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAccountNoBudget(t *testing.T) {
	recorder := test.Request(t, http.MethodPost, "http://example.com/v1/accounts", models.AccountCreate{BudgetID: uuid.New()})
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestGetAccount(t *testing.T) {
	a := createTestAccount(t, models.AccountCreate{})

	r := test.Request(t, http.MethodGet, a.Data.Links.Self, "")
	assert.Equal(t, http.StatusOK, r.Code)
}

func TestGetAccountTransactionsNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, http.MethodGet, "http://example.com/v1/accounts/57372/transactions", "")
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestUpdateAccount(t *testing.T) {
	a := createTestAccount(t, models.AccountCreate{Name: "Original name"})

	r := test.Request(t, http.MethodPatch, a.Data.Links.Self, models.AccountCreate{Name: "Updated new account for testing"})
	test.AssertHTTPStatus(t, http.StatusOK, &r)

	var u controllers.AccountResponse
	test.DecodeResponse(t, &r, &u)

	assert.Equal(t, "Updated new account for testing", u.Data.Name)
}

func TestUpdateAccountBroken(t *testing.T) {
	a := createTestAccount(t, models.AccountCreate{
		Name: "New Account",
		Note: "More tests something something",
	})

	r := test.Request(t, http.MethodPatch, a.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestUpdateNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, http.MethodPatch, "http://example.com/v1/accounts/9b81de41-eead-451d-bc6b-31fceedd236c", models.AccountCreate{Name: "This account does not exist"})
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAccountsAndEmptyList(t *testing.T) {
	r := test.Request(t, http.MethodGet, "http://example.com/v1/accounts", "")

	var l controllers.AccountListResponse
	test.DecodeResponse(t, &r, &l)

	for _, a := range l.Data {
		r = test.Request(t, http.MethodDelete, a.Links.Self, "")
		test.AssertHTTPStatus(t, http.StatusNoContent, &r)
	}

	r = test.Request(t, http.MethodGet, "http://example.com/v1/accounts", "")
	test.DecodeResponse(t, &r, &l)

	// Verify that the account list is an empty list, not null
	assert.NotNil(t, l.Data)
	assert.Empty(t, l.Data)
}

func TestDeleteNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, http.MethodDelete, "http://example.com/v1/accounts/77b70a75-4bb3-4d1d-90cf-5b7a61f452f5", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAccountWithBody(t *testing.T) {
	a := createTestAccount(t, models.AccountCreate{Name: "Delete me now!"})

	r := test.Request(t, http.MethodDelete, a.Data.Links.Self, models.AccountCreate{Name: "Some other account"})
	test.AssertHTTPStatus(t, http.StatusNoContent, &r)

	r = test.Request(t, http.MethodGet, a.Data.Links.Self, "")
}
