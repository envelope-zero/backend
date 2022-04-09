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

type TransactionListResponse struct {
	test.APIResponse
	Data []models.Transaction
}

type TransactionDetailResponse struct {
	test.APIResponse
	Data models.Transaction
}

func TestGetTransactions(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/transactions", "")

	var response TransactionListResponse
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 3) {
		assert.FailNow(t, "Response does not have exactly 3 items")
	}

	januaryTransaction := response.Data[0]
	assert.Equal(t, uint64(1), januaryTransaction.BudgetID)
	assert.Equal(t, "Water bill for January", januaryTransaction.Note)
	assert.Equal(t, true, januaryTransaction.Reconciled)
	assert.Equal(t, uint64(1), januaryTransaction.SourceAccountID)
	assert.Equal(t, uint64(3), januaryTransaction.DestinationAccountID)
	assert.Equal(t, uint64(1), januaryTransaction.EnvelopeID)
	if !decimal.NewFromFloat(10).Equal(januaryTransaction.Amount) {
		assert.Fail(t, "Transaction amount does not equal 10", januaryTransaction.Amount)
	}

	februaryTransaction := response.Data[1]
	assert.Equal(t, uint64(1), februaryTransaction.BudgetID)
	assert.Equal(t, "Water bill for February", februaryTransaction.Note)
	assert.Equal(t, false, februaryTransaction.Reconciled)
	assert.Equal(t, uint64(1), februaryTransaction.SourceAccountID)
	assert.Equal(t, uint64(3), februaryTransaction.DestinationAccountID)
	assert.Equal(t, uint64(1), februaryTransaction.EnvelopeID)
	if !decimal.NewFromFloat(5).Equal(februaryTransaction.Amount) {
		assert.Fail(t, "Transaction amount does not equal 5", februaryTransaction.Amount)
	}

	marchTransaction := response.Data[2]
	assert.Equal(t, uint64(1), marchTransaction.BudgetID)
	assert.Equal(t, "Water bill for March", marchTransaction.Note)
	assert.Equal(t, false, marchTransaction.Reconciled)
	assert.Equal(t, uint64(1), marchTransaction.SourceAccountID)
	assert.Equal(t, uint64(3), marchTransaction.DestinationAccountID)
	assert.Equal(t, uint64(1), marchTransaction.EnvelopeID)
	if !decimal.NewFromFloat(15).Equal(marchTransaction.Amount) {
		assert.Fail(t, "Transaction amount does not equal 15", marchTransaction.Amount)
	}

	for _, transaction := range response.Data {
		diff := time.Now().Sub(transaction.CreatedAt)
		assert.LessOrEqual(t, diff, test.TOLERANCE)

		diff = time.Now().Sub(transaction.UpdatedAt)
		assert.LessOrEqual(t, diff, test.TOLERANCE)
	}
}

func TestNoTransactionNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/transactions/37", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", `{ "note": "More tests something something", "amount": 1253.17 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiTransaction TransactionDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiTransaction)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbTransaction models.Transaction
	models.DB.First(&dbTransaction, apiTransaction.Data.ID)

	// Set the balance to 0 to compare to the database object
	apiTransaction.Data.Amount = decimal.NewFromFloat(0)
	dbTransaction.Amount = decimal.NewFromFloat(0)
	assert.Equal(t, dbTransaction, apiTransaction.Data)
}

func TestCreateTransactionNoAmount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", `{ "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateBrokenTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", `{ "createdAt": "New Transaction", "note": "More tests for transactions to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateNegativeAmountTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", `{ "amount": -17.12, "note": "Negative amounts are not allowed, this must fail" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateTransactionNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetTransaction(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/transactions/1", "")
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var transaction TransactionDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&transaction)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbTransaction models.Transaction
	models.DB.First(&dbTransaction, transaction.Data.ID)

	if !decimal.NewFromFloat(10).Equals(transaction.Data.Amount) {
		assert.Fail(t, "Transaction amount does not equal 10", transaction.Data.Amount)
	}

	// Set the balance to 0 to compare to the database object
	transaction.Data.Amount = decimal.NewFromFloat(0)
	dbTransaction.Amount = decimal.NewFromFloat(0)
	assert.Equal(t, dbTransaction, transaction.Data)
}

func TestUpdateTransaction(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", `{ "note": "More tests something something", "amount": 584.42 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var transaction TransactionDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&transaction)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/transactions/%v", transaction.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "note": "Updated new transaction for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedTransaction TransactionDetailResponse
	err = json.NewDecoder(recorder.Body).Decode(&updatedTransaction)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, "Updated new transaction for testing", updatedTransaction.Data.Note)
}

func TestUpdateTransactionBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", `{ "amount": 5883.53, "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var transaction TransactionDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&transaction)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/transactions/%v", transaction.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "note": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateTransactionNegativeAmount(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", `{ "amount": 382.18 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var transaction TransactionDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&transaction)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/transactions/%v", transaction.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "amount": -58.23 }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingTransaction(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/1/transactions/48902805", `{ "note": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteTransaction(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/transactions/1", "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingTransaction(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/transactions/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteTransactionWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/transactions", `{ "name": "Delete me now!", "amount": 17.21 }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var transaction TransactionDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&transaction)
	if err != nil {
		assert.Fail(t, "Parsing error", "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/transactions/%v", transaction.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
