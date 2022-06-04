package controllers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func createTestTransaction(t *testing.T, c models.TransactionCreate) controllers.TransactionResponse {
	r := test.Request(t, "POST", "http://example.com/v1/transactions", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var tr controllers.TransactionResponse
	test.DecodeResponse(t, &r, &tr)

	return tr
}

func (suite *TestSuiteEnv) TestGetTransactions() {
	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/transactions", "")

	var response controllers.TransactionListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	if !assert.Len(suite.T(), response.Data, 3) {
		assert.FailNow(suite.T(), "Response does not have exactly 3 items")
	}

	januaryTransaction := response.Data[0]
	assert.Equal(suite.T(), "Water bill for January", januaryTransaction.Note)
	assert.Equal(suite.T(), true, januaryTransaction.Reconciled)
	if !decimal.NewFromFloat(10).Equal(januaryTransaction.Amount) {
		assert.Fail(suite.T(), "Transaction amount does not equal 10", januaryTransaction.Amount)
	}

	februaryTransaction := response.Data[1]
	assert.Equal(suite.T(), "Water bill for February", februaryTransaction.Note)
	assert.Equal(suite.T(), false, februaryTransaction.Reconciled)
	if !decimal.NewFromFloat(5).Equal(februaryTransaction.Amount) {
		assert.Fail(suite.T(), "Transaction amount does not equal 5", februaryTransaction.Amount)
	}

	marchTransaction := response.Data[2]
	assert.Equal(suite.T(), "Water bill for March", marchTransaction.Note)
	assert.Equal(suite.T(), false, marchTransaction.Reconciled)
	if !decimal.NewFromFloat(15).Equal(marchTransaction.Amount) {
		assert.Fail(suite.T(), "Transaction amount does not equal 15", marchTransaction.Amount)
	}

	for _, transaction := range response.Data {
		diff := time.Since(transaction.CreatedAt)
		assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)

		diff = time.Since(transaction.UpdatedAt)
		assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)
	}
}

func (suite *TestSuiteEnv) TestNoTransactionNotFound() {
	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/transactions/048b061f-3b6b-45ab-b0e9-0f38d2fff0c8", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestTransactionInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/transactions/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/transactions/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/transactions/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/transactions/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/transactions/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/transactions/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/transactions/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateTransaction() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "note": "More tests something something", "amount": 1253.17 }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var apiTransaction controllers.TransactionResponse
	test.DecodeResponse(suite.T(), &recorder, &apiTransaction)

	var dbTransaction models.Transaction
	database.DB.First(&dbTransaction, apiTransaction.Data.ID)

	assert.True(suite.T(), apiTransaction.Data.Amount.Equal(dbTransaction.Amount))
}

func (suite *TestSuiteEnv) TestCreateTransactionNoAmount() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateBrokenTransaction() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "createdAt": "New Transaction", "note": "More tests for transactions to ensure less brokenness something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateNegativeAmountTransaction() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "amount": -17.12, "note": "Negative amounts are not allowed, this must fail" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateNonExistingBudgetTransaction() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "budgetId": "978e95a0-90f2-4dee-91fd-ee708c30301c", "amount": 32.12, "note": "The budget with this id must exist, so this must fail" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestCreateTransactionNoBody() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestGetTransaction() {
	tr := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(13.71)})

	r := test.Request(suite.T(), http.MethodGet, tr.Data.Links.Self, "")
	assert.Equal(suite.T(), http.StatusOK, r.Code)
}

func (suite *TestSuiteEnv) TestUpdateTransaction() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "note": "More tests something something", "amount": 584.42 }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var transaction controllers.TransactionResponse
	test.DecodeResponse(suite.T(), &recorder, &transaction)

	recorder = test.Request(suite.T(), "PATCH", transaction.Data.Links.Self, `{ "note": "Updated new transaction for testing" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedTransaction controllers.TransactionResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedTransaction)

	assert.Equal(suite.T(), "Updated new transaction for testing", updatedTransaction.Data.Note)
}

func (suite *TestSuiteEnv) TestUpdateTransactionBroken() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "amount": 5883.53, "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var transaction controllers.TransactionResponse
	test.DecodeResponse(suite.T(), &recorder, &transaction)

	recorder = test.Request(suite.T(), "PATCH", transaction.Data.Links.Self, `{ "note": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateTransactionNegativeAmount() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "amount": 382.18 }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var transaction controllers.TransactionResponse
	test.DecodeResponse(suite.T(), &recorder, &transaction)

	recorder = test.Request(suite.T(), "PATCH", transaction.Data.Links.Self, `{ "amount": -58.23 }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingTransaction() {
	recorder := test.Request(suite.T(), "PATCH", "http://example.com/v1/transactions/6ae3312c-23cf-4225-9a81-4f218ba41b00", `{ "note": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteTransaction() {
	tr := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(123.12)})

	recorder := test.Request(suite.T(), "DELETE", tr.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingTransaction() {
	recorder := test.Request(suite.T(), "DELETE", "http://example.com/v1/transactions/4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteTransactionWithBody() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", `{ "name": "Delete me now!", "amount": 17.21 }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var transaction controllers.TransactionResponse
	test.DecodeResponse(suite.T(), &recorder, &transaction)

	recorder = test.Request(suite.T(), "DELETE", transaction.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}
