package controllers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/controllers"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func createTestTransaction(t *testing.T, c models.TransactionCreate) controllers.TransactionResponse {
	r := test.Request(t, "POST", "/v1/transactions", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var tr controllers.TransactionResponse
	test.DecodeResponse(t, &r, &tr)

	return tr
}

func TestGetTransactions(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/transactions", "")

	var response controllers.TransactionListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 3) {
		assert.FailNow(t, "Response does not have exactly 3 items")
	}

	januaryTransaction := response.Data[0]
	assert.Equal(t, "Water bill for January", januaryTransaction.Note)
	assert.Equal(t, true, januaryTransaction.Reconciled)
	if !decimal.NewFromFloat(10).Equal(januaryTransaction.Amount) {
		assert.Fail(t, "Transaction amount does not equal 10", januaryTransaction.Amount)
	}

	februaryTransaction := response.Data[1]
	assert.Equal(t, "Water bill for February", februaryTransaction.Note)
	assert.Equal(t, false, februaryTransaction.Reconciled)
	if !decimal.NewFromFloat(5).Equal(februaryTransaction.Amount) {
		assert.Fail(t, "Transaction amount does not equal 5", februaryTransaction.Amount)
	}

	marchTransaction := response.Data[2]
	assert.Equal(t, "Water bill for March", marchTransaction.Note)
	assert.Equal(t, false, marchTransaction.Reconciled)
	if !decimal.NewFromFloat(15).Equal(marchTransaction.Amount) {
		assert.Fail(t, "Transaction amount does not equal 15", marchTransaction.Amount)
	}

	for _, transaction := range response.Data {
		diff := time.Since(transaction.CreatedAt)
		assert.LessOrEqual(t, diff, test.TOLERANCE)

		diff = time.Since(transaction.UpdatedAt)
		assert.LessOrEqual(t, diff, test.TOLERANCE)
	}
}

func TestNoTransactionNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/transactions/048b061f-3b6b-45ab-b0e9-0f38d2fff0c8", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestTransactionInvalidIDs(t *testing.T) {
	/*
	 *  GET
	 */
	r := test.Request(t, http.MethodGet, "/v1/transactions/-56", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/transactions/notANumber", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/transactions/23", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(t, http.MethodPatch, "/v1/transactions/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodPatch, "/v1/transactions/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(t, http.MethodDelete, "/v1/transactions/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodDelete, "/v1/transactions/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "note": "More tests something something", "amount": 1253.17 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiTransaction controllers.TransactionResponse
	test.DecodeResponse(t, &recorder, &apiTransaction)

	var dbTransaction models.Transaction
	models.DB.First(&dbTransaction, apiTransaction.Data.ID)

	assert.True(t, apiTransaction.Data.Amount.Equal(dbTransaction.Amount))
}

func TestCreateTransactionNoAmount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateBrokenTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "createdAt": "New Transaction", "note": "More tests for transactions to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateNegativeAmountTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "amount": -17.12, "note": "Negative amounts are not allowed, this must fail" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateNonExistingBudgetTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "budgetId": "978e95a0-90f2-4dee-91fd-ee708c30301c", "amount": 32.12, "note": "The budget with this id must exist, so this must fail" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateTransactionNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetTransaction(t *testing.T) {
	tr := createTestTransaction(t, models.TransactionCreate{Amount: decimal.NewFromFloat(13.71)})

	r := test.Request(t, http.MethodGet, tr.Data.Links.Self, "")
	assert.Equal(t, http.StatusOK, r.Code)
}

func TestUpdateTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "note": "More tests something something", "amount": 584.42 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var transaction controllers.TransactionResponse
	test.DecodeResponse(t, &recorder, &transaction)

	recorder = test.Request(t, "PATCH", transaction.Data.Links.Self, `{ "note": "Updated new transaction for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedTransaction controllers.TransactionResponse
	test.DecodeResponse(t, &recorder, &updatedTransaction)

	assert.Equal(t, "Updated new transaction for testing", updatedTransaction.Data.Note)
}

func TestUpdateTransactionBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "amount": 5883.53, "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var transaction controllers.TransactionResponse
	test.DecodeResponse(t, &recorder, &transaction)

	recorder = test.Request(t, "PATCH", transaction.Data.Links.Self, `{ "note": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateTransactionNegativeAmount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "amount": 382.18 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var transaction controllers.TransactionResponse
	test.DecodeResponse(t, &recorder, &transaction)

	recorder = test.Request(t, "PATCH", transaction.Data.Links.Self, `{ "amount": -58.23 }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingTransaction(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/transactions/6ae3312c-23cf-4225-9a81-4f218ba41b00", `{ "note": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteTransaction(t *testing.T) {
	tr := createTestTransaction(t, models.TransactionCreate{Amount: decimal.NewFromFloat(123.12)})

	recorder := test.Request(t, "DELETE", tr.Data.Links.Self, "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingTransaction(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/transactions/4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteTransactionWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/transactions", `{ "name": "Delete me now!", "amount": 17.21 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var transaction controllers.TransactionResponse
	test.DecodeResponse(t, &recorder, &transaction)

	recorder = test.Request(t, "DELETE", transaction.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
