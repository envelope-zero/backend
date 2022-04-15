package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type AccountListResponse struct {
	test.APIResponse
	Data []models.Account
}

type AccountDetailResponse struct {
	test.APIResponse
	Data models.Account
}

func TestGetAccounts(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/accounts", "")

	var response AccountListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 3) {
		assert.FailNow(t, "Response does not have exactly 3 items")
	}

	bankAccount := response.Data[0]
	assert.Equal(t, uint64(1), bankAccount.BudgetID)
	assert.Equal(t, "Bank Account", bankAccount.Name)
	assert.Equal(t, true, bankAccount.OnBudget)
	assert.Equal(t, false, bankAccount.External)

	cashAccount := response.Data[1]
	assert.Equal(t, uint64(1), cashAccount.BudgetID)
	assert.Equal(t, "Cash Account", cashAccount.Name)
	assert.Equal(t, false, cashAccount.OnBudget)
	assert.Equal(t, false, cashAccount.External)

	externalAccount := response.Data[2]
	assert.Equal(t, uint64(1), externalAccount.BudgetID)
	assert.Equal(t, "External Account", externalAccount.Name)
	assert.Equal(t, false, externalAccount.OnBudget)
	assert.Equal(t, true, externalAccount.External)

	for _, account := range response.Data {
		diff := time.Now().Sub(account.CreatedAt)
		assert.LessOrEqual(t, diff, test.TOLERANCE)

		diff = time.Now().Sub(account.UpdatedAt)
		assert.LessOrEqual(t, diff, test.TOLERANCE)
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
	recorder := test.Request(t, "GET", "/v1/budgets/1/accounts/37", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

// TestAccountInvalidIDs verifies that on non-number requests for account IDs,
// the API returs a Bad Request status code.
func TestAccountInvalidIDs(t *testing.T) {
	r := test.Request(t, "GET", "/v1/budgets/1/accounts/-56", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "GET", "/v1/budgets/1/accounts/notANumber", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "GET", "/v1/budgets/-61/accounts/56", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, "GET", "/v1/budgets/RandomStringThatIsNotAUint64/accounts/1", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

// TestNonexistingBudgetAccounts404 is a regression test for https://github.com/envelope-zero/backend/issues/89.
//
// It verifies that for a non-existing budget, the accounts endpoint raises a 404
// instead of returning an empty list.
func TestNonexistingBudgetAccounts404(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/999/accounts", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

// TestAccountParentChecked is a regression test for https://github.com/envelope-zero/backend/issues/90.
//
// It verifies that the account details endpoint for a budget only returns accounts that belong to the
// budget.
func TestAccountParentChecked(t *testing.T) {
	r := test.Request(t, "POST", "/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var budget BudgetDetailResponse
	test.DecodeResponse(t, &r, &budget)

	path := fmt.Sprintf("/v1/budgets/%v", budget.Data.ID)
	r = test.Request(t, "GET", path+"/accounts/1", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &r)

	r = test.Request(t, "DELETE", path, "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &r)
}

func TestCreateAccount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "name": "New Account", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiAccount AccountDetailResponse
	test.DecodeResponse(t, &recorder, &apiAccount)

	var dbAccount models.Account
	models.DB.First(&dbAccount, apiAccount.Data.ID)
}

func TestCreateBrokenAccount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "createdAt": "New Account", "note": "More tests for accounts to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateAccountNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetAccount(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/accounts/1", "")
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var account AccountDetailResponse
	test.DecodeResponse(t, &recorder, &account)

	var dbAccount models.Account
	models.DB.First(&dbAccount, account.Data.ID)

	// The test transactions have a sum of -30
	if !decimal.NewFromFloat(-30).Equals(account.Data.Balance) {
		assert.Fail(t, "Account balance does not equal -30", account.Data.Balance)
	}
}

func TestGetAccountTransactions(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/accounts/1/transactions", "")

	var response TransactionListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, response.Data, 3)
}

func TestGetAccountTransactionsNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/accounts/57372/transactions", "")
	assert.Equal(t, 404, recorder.Code)
}

func TestUpdateAccount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "name": "New Account", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var account AccountDetailResponse
	test.DecodeResponse(t, &recorder, &account)

	path := fmt.Sprintf("/v1/budgets/1/accounts/%v", account.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new account for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedAccount AccountDetailResponse
	test.DecodeResponse(t, &recorder, &updatedAccount)

	assert.Equal(t, "Updated new account for testing", updatedAccount.Data.Name)
}

func TestUpdateAccountBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "name": "New Account", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var account AccountDetailResponse
	test.DecodeResponse(t, &recorder, &account)

	path := fmt.Sprintf("/v1/budgets/1/accounts/%v", account.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/1/accounts/48902805", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAccountsAndEmptyList(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/accounts/1", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)

	recorder = test.Request(t, "DELETE", "/v1/budgets/1/accounts/2", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)

	recorder = test.Request(t, "DELETE", "/v1/budgets/1/accounts/3", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)

	recorder = test.Request(t, "DELETE", "/v1/budgets/1/accounts/4", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)

	recorder = test.Request(t, "DELETE", "/v1/budgets/1/accounts/5", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)

	recorder = test.Request(t, "DELETE", "/v1/budgets/1/accounts/6", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)

	recorder = test.Request(t, "GET", "/v1/budgets/1/accounts", "")
	var apiResponse AccountListResponse
	test.DecodeResponse(t, &recorder, &apiResponse)

	// Verify that the account list is an empty list, not null
	assert.NotNil(t, apiResponse.Data)
	assert.Empty(t, apiResponse.Data)
}

func TestDeleteNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/accounts/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAccountWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var account AccountDetailResponse
	test.DecodeResponse(t, &recorder, &account)

	path := fmt.Sprintf("/v1/budgets/1/accounts/%v", account.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
