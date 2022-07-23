package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func createTestTransaction(t *testing.T, c models.TransactionCreate) controllers.TransactionResponse {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = createTestBudget(t, models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	if c.SourceAccountID == uuid.Nil {
		c.SourceAccountID = createTestAccount(t, models.AccountCreate{Name: "Source Account"}).Data.ID
	}

	if c.DestinationAccountID == uuid.Nil {
		c.DestinationAccountID = createTestAccount(t, models.AccountCreate{Name: "Destination Account"}).Data.ID
	}

	if c.EnvelopeID == uuid.Nil {
		c.EnvelopeID = createTestEnvelope(t, models.EnvelopeCreate{Name: "Transaction Test Envelope"}).Data.ID
	}

	r := test.Request(t, "POST", "http://example.com/v1/transactions", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var tr controllers.TransactionResponse
	test.DecodeResponse(t, &r, &tr)

	return tr
}

func (suite *TestSuiteEnv) TestGetTransactions() {
	_ = createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(17.23)})
	_ = createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(23.42)})

	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/transactions", "")

	var response controllers.TransactionListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 2)
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
	_ = createTestTransaction(suite.T(), models.TransactionCreate{Note: "More tests something something", Amount: decimal.NewFromFloat(1253.17)})
}

func (suite *TestSuiteEnv) TestCreateTransactionMissingReference() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})
	category := createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	account := createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID})

	// Missing Budget
	r := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.Transaction{
		TransactionCreate: models.TransactionCreate{
			SourceAccountID:      account.Data.ID,
			DestinationAccountID: account.Data.ID,
			EnvelopeID:           envelope.Data.ID,
		},
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	// Missing Envelope
	r = test.Request(suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.Data.ID,
			SourceAccountID:      account.Data.ID,
			DestinationAccountID: account.Data.ID,
		},
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	// Missing Source Account
	r = test.Request(suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.Data.ID,
			DestinationAccountID: account.Data.ID,
			EnvelopeID:           envelope.Data.ID,
		},
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	// Missing Destination Account
	r = test.Request(suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:        budget.Data.ID,
			SourceAccountID: account.Data.ID,
			EnvelopeID:      envelope.Data.ID,
		},
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
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
	budget := createTestBudget(suite.T(), models.BudgetCreate{})
	category := createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	account := createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID})

	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/transactions", models.TransactionCreate{
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: account.Data.ID,
		EnvelopeID:           envelope.Data.ID,
		Amount:               decimal.NewFromFloat(-17.12),
		Note:                 "Negative amounts are not allowed, this must fail",
	})

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
	transaction := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(584.42), Note: "Test note for transaction"})

	recorder := test.Request(suite.T(), "PATCH", transaction.Data.Links.Self, map[string]any{
		"note": "",
	})
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedTransaction controllers.TransactionResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedTransaction)

	assert.Equal(suite.T(), "", updatedTransaction.Data.Note)
}

func (suite *TestSuiteEnv) TestUpdateTransactionSourceDestinationEqual() {
	transaction := createTestTransaction(suite.T(), models.TransactionCreate{Note: "More tests something something", Amount: decimal.NewFromFloat(1253.17)})

	r := test.Request(suite.T(), http.MethodPatch, transaction.Data.Links.Self, map[string]any{
		"destinationAccountId": transaction.Data.SourceAccountID,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestUpdateTransactionBrokenJSON() {
	transaction := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(5883.53)})

	recorder := test.Request(suite.T(), "PATCH", transaction.Data.Links.Self, `{ "amount": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateTransactionInvalidType() {
	transaction := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(5883.53)})

	recorder := test.Request(suite.T(), http.MethodPatch, transaction.Data.Links.Self, map[string]any{
		"amount": false,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateTransactionInvalidBudgetID() {
	transaction := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(5883.53)})

	// Sets the BudgetID to uuid.Nil
	recorder := test.Request(suite.T(), http.MethodPatch, transaction.Data.Links.Self, models.TransactionCreate{})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateTransactionNegativeAmount() {
	transaction := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(382.18)})

	recorder := test.Request(suite.T(), "PATCH", transaction.Data.Links.Self, `{ "amount": -58.23 }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingTransaction() {
	recorder := test.Request(suite.T(), "PATCH", "http://example.com/v1/transactions/6ae3312c-23cf-4225-9a81-4f218ba41b00", `{ "note": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteTransaction() {
	transaction := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(123.12)})

	recorder := test.Request(suite.T(), "DELETE", transaction.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingTransaction() {
	recorder := test.Request(suite.T(), "DELETE", "http://example.com/v1/transactions/4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteTransactionWithBody() {
	transaction := createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(17.21)})
	recorder := test.Request(suite.T(), "DELETE", transaction.Data.Links.Self, `{ "amount": "23.91" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteNullTransaction() {
	r := test.Request(suite.T(), "DELETE", "http://example.com/v1/transactions/00000000-0000-0000-0000-000000000000", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}
