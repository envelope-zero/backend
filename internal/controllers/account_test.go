package controllers_test

import (
	"encoding/json"
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
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, response.Data, 3)

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
}

func TestNoAccountNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/accounts/37", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateAccount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "name": "New Account", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiAccount AccountDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiAccount)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbAccount models.Account
	models.DB.First(&dbAccount, apiAccount.Data.ID)

	// Set the balance to 0 to compare to the database object
	apiAccount.Data.Balance = decimal.NewFromFloat(0)
	dbAccount.Balance = decimal.NewFromFloat(0)

	assert.Equal(t, dbAccount, apiAccount.Data)
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
	err := json.NewDecoder(recorder.Body).Decode(&account)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbAccount models.Account
	models.DB.First(&dbAccount, account.Data.ID)

	// The test transactions have a sum of -30
	if !decimal.NewFromFloat(-30).Equals(account.Data.Balance) {
		assert.Fail(t, "Account balance does not equal -30", account.Data.Balance)
	}

	// Set the balance to 0 to compare to the database object
	account.Data.Balance = decimal.NewFromFloat(0)
	dbAccount.Balance = decimal.NewFromFloat(0)
	assert.Equal(t, dbAccount, account.Data)
}

func TestGetAccountTransactions(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/accounts/1/transactions", "")

	var response TransactionListResponse
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, response.Data, 3)
}

func TestGetAccountTransactionsNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/accounts/57372/transactions", "")

	var response TransactionListResponse
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, 404, recorder.Code)
	assert.Len(t, response.Data, 0)
}

func TestUpdateAccount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "name": "New Account", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var account AccountDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&account)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/accounts/%v", account.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new account for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedAccount AccountDetailResponse
	err = json.NewDecoder(recorder.Body).Decode(&updatedAccount)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, "Updated new account for testing", updatedAccount.Data.Name)
}

func TestUpdateAccountBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "name": "New Account", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var account AccountDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&account)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/accounts/%v", account.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/1/accounts/48902805", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAccount(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/accounts/1", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingAccount(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/accounts/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteAccountWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/accounts", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var account AccountDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&account)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/accounts/%v", account.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
