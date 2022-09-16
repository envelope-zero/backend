package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestTransaction(t *testing.T, c models.TransactionCreate) controllers.TransactionResponse {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = suite.createTestBudget(t, models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	if c.SourceAccountID == uuid.Nil {
		c.SourceAccountID = suite.createTestAccount(t, models.AccountCreate{Name: "Source Account"}).Data.ID
	}

	if c.DestinationAccountID == uuid.Nil {
		c.DestinationAccountID = suite.createTestAccount(t, models.AccountCreate{Name: "Destination Account"}).Data.ID
	}

	if c.EnvelopeID == &uuid.Nil {
		*c.EnvelopeID = suite.createTestEnvelope(t, models.EnvelopeCreate{Name: "Transaction Test Envelope"}).Data.ID
	}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v1/transactions", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var tr controllers.TransactionResponse
	test.DecodeResponse(t, &r, &tr)

	return tr
}

func (suite *TestSuiteStandard) TestOptionsTransaction() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v1/transactions", uuid.New())
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/transactions/NotParseableAsUUID", "")
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(31)}).Data.Links.Self
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestGetTransactions() {
	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(17.23)})
	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(23.42)})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/transactions", "")

	var response controllers.TransactionListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 2)
}

func (suite *TestSuiteStandard) TestGetTransactionsInvalidQuery() {
	tests := []string{
		"budget=DefinitelyACat",
		"source=MaybeADog",
		"destination=OrARat?",
		"envelope=NopeDefinitelyAMole",
		"date=A long time ago",
		"amount=Seventeen Cents",
		"reconciled=I don't think so",
		"account=ItIsAHippo!",
	}

	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v1/transactions?%s", tt), "")
			test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
		})
	}
}

func (suite *TestSuiteStandard) TestGetTransactionsFilter() {
	b := suite.createTestBudget(suite.T(), models.BudgetCreate{})

	a1 := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: b.Data.ID})
	a2 := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: b.Data.ID})
	a3 := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: b.Data.ID})

	c := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: b.Data.ID})

	e1 := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: c.Data.ID})
	e2 := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: c.Data.ID})

	e1ID := &e1.Data.ID
	e2ID := &e2.Data.ID

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2018, 9, 5, 17, 13, 29, 45256, time.UTC),
		Amount:               decimal.NewFromFloat(2.718),
		Note:                 "This was an important expense",
		BudgetID:             b.Data.ID,
		EnvelopeID:           e1ID,
		SourceAccountID:      a1.Data.ID,
		DestinationAccountID: a2.Data.ID,
		Reconciled:           false,
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2016, 5, 1, 14, 13, 25, 584575, time.UTC),
		Amount:               decimal.NewFromFloat(11235.813),
		Note:                 "Not important",
		BudgetID:             b.Data.ID,
		EnvelopeID:           e2ID,
		SourceAccountID:      a2.Data.ID,
		DestinationAccountID: a1.Data.ID,
		Reconciled:           false,
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2021, 2, 6, 5, 1, 0, 585, time.UTC),
		Amount:               decimal.NewFromFloat(2.718),
		Note:                 "",
		BudgetID:             b.Data.ID,
		EnvelopeID:           e1ID,
		SourceAccountID:      a3.Data.ID,
		DestinationAccountID: a2.Data.ID,
		Reconciled:           true,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Exact Date", fmt.Sprintf("date=%s", time.Date(2021, 2, 6, 5, 1, 0, 585, time.UTC).Format(time.RFC3339Nano)), 1},
		{"Exact Amount", fmt.Sprintf("amount=%s", decimal.NewFromFloat(2.718).String()), 2},
		{"Note", "note=Not important", 1},
		{"No note", "note=", 1},
		{"Budget Match", fmt.Sprintf("budget=%s", b.Data.ID), 3},
		{"Envelope 2", fmt.Sprintf("envelope=%s", e2.Data.ID), 1},
		{"Non-existing Source Account", "source=3340a084-acf8-4cb4-8f86-9e7f88a86190", 0},
		{"Destination Account", fmt.Sprintf("destination=%s", a2.Data.ID), 2},
		{"Reconciled", "reconciled=false", 2},
		{"Non-existing Account", "account=534a3562-c5e8-46d1-a2e2-e96c00e7efec", 0},
		{"Existing Account 2", fmt.Sprintf("account=%s", a2.Data.ID), 3},
		{"Existing Account 1", fmt.Sprintf("account=%s", a1.Data.ID), 2},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.TransactionListResponse
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v1/transactions?%s", tt.query), "")
			test.AssertHTTPStatus(t, http.StatusOK, &r)
			test.DecodeResponse(t, &r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestNoTransactionNotFound() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/transactions/048b061f-3b6b-45ab-b0e9-0f38d2fff0c8", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestTransactionInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/transactions/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/transactions/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/transactions/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/transactions/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/transactions/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/transactions/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/transactions/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteStandard) TestCreateTransaction() {
	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{Note: "More tests something something", Amount: decimal.NewFromFloat(1253.17)})
}

func (suite *TestSuiteStandard) TestTransactionSorting() {
	tFebrurary := suite.createTestTransaction(suite.T(), models.TransactionCreate{Note: "Should be second in the list", Amount: decimal.NewFromFloat(1253.17), Date: time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC)})

	tMarch := suite.createTestTransaction(suite.T(), models.TransactionCreate{Note: "Should be first in the list", Amount: decimal.NewFromFloat(1253.17), Date: time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC)})

	tJanuary := suite.createTestTransaction(suite.T(), models.TransactionCreate{Note: "Should be third in the list", Amount: decimal.NewFromFloat(1253.17), Date: time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/transactions", "")
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)

	var transactions controllers.TransactionListResponse
	test.DecodeResponse(suite.T(), &r, &transactions)

	if !assert.Len(suite.T(), transactions.Data, 3, "There are not exactly three transactions") {
		assert.FailNow(suite.T(), "Number of transactions is wrong, aborting")
	}
	assert.Equal(suite.T(), tMarch.Data.Date, transactions.Data[0].Date, "The first transaction is not the March transaction")
	assert.Equal(suite.T(), tFebrurary.Data.Date, transactions.Data[1].Date, "The second transaction is not the February transaction")
	assert.Equal(suite.T(), tJanuary.Data.Date, transactions.Data[2].Date, "The third transaction is not the January transaction")
}

func (suite *TestSuiteStandard) TestCreateTransactionMissingReference() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	account := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID})

	// Missing Budget
	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.Transaction{
		TransactionCreate: models.TransactionCreate{
			SourceAccountID:      account.Data.ID,
			DestinationAccountID: account.Data.ID,
			EnvelopeID:           &envelope.Data.ID,
		},
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	// Missing Envelope
	r = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.Data.ID,
			SourceAccountID:      account.Data.ID,
			DestinationAccountID: account.Data.ID,
		},
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	// Missing Source Account
	r = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.Data.ID,
			DestinationAccountID: account.Data.ID,
			EnvelopeID:           &envelope.Data.ID,
		},
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	// Missing Destination Account
	r = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:        budget.Data.ID,
			SourceAccountID: account.Data.ID,
			EnvelopeID:      &envelope.Data.ID,
		},
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteStandard) TestCreateTransactionNoAmount() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", `{ "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestCreateBrokenTransaction() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", `{ "createdAt": "New Transaction", "note": "More tests for transactions to ensure less brokenness something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestCreateNegativeAmountTransaction() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	account := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", models.TransactionCreate{
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: account.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Amount:               decimal.NewFromFloat(-17.12),
		Note:                 "Negative amounts are not allowed, this must fail",
	})

	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestCreateNonExistingBudgetTransaction() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", `{ "budgetId": "978e95a0-90f2-4dee-91fd-ee708c30301c", "amount": 32.12, "note": "The budget with this id must exist, so this must fail" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestCreateNoEnvelopeTransactionTransfer() {
	c := models.TransactionCreate{
		BudgetID:             suite.createTestBudget(suite.T(), models.BudgetCreate{Name: "Testing budget for transfer"}).Data.ID,
		SourceAccountID:      suite.createTestAccount(suite.T(), models.AccountCreate{Name: "Internal Source Account", External: false}).Data.ID,
		DestinationAccountID: suite.createTestAccount(suite.T(), models.AccountCreate{Name: "Internal destination account", External: false}).Data.ID,
		Amount:               decimal.NewFromFloat(500),
	}

	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", c)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)
}

func (suite *TestSuiteStandard) TestCreateNoEnvelopeTransactionOutgoing() {
	c := models.TransactionCreate{
		BudgetID:             suite.createTestBudget(suite.T(), models.BudgetCreate{Name: "Testing budget for transfer"}).Data.ID,
		SourceAccountID:      suite.createTestAccount(suite.T(), models.AccountCreate{Name: "Internal Source Account", External: false}).Data.ID,
		DestinationAccountID: suite.createTestAccount(suite.T(), models.AccountCreate{Name: "External destination account", External: true}).Data.ID,
		Amount:               decimal.NewFromFloat(350),
	}

	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", c)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)
}

func (suite *TestSuiteStandard) TestCreateNonExistingEnvelopeTransactionTransfer() {
	id := uuid.New()

	c := models.TransactionCreate{
		BudgetID:             suite.createTestBudget(suite.T(), models.BudgetCreate{Name: "Testing budget for transfer"}).Data.ID,
		SourceAccountID:      suite.createTestAccount(suite.T(), models.AccountCreate{Name: "Internal Source Account", External: false}).Data.ID,
		DestinationAccountID: suite.createTestAccount(suite.T(), models.AccountCreate{Name: "External destination account", External: true}).Data.ID,
		Amount:               decimal.NewFromFloat(350),
		EnvelopeID:           &id,
	}

	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", c)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestCreateTransactionNoBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/transactions", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestGetTransaction() {
	tr := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(13.71)})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, tr.Data.Links.Self, "")
	assert.Equal(suite.T(), http.StatusOK, r.Code)
}

func (suite *TestSuiteStandard) TestUpdateTransaction() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(584.42), Note: "Test note for transaction"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, map[string]any{
		"note": "",
	})
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedTransaction controllers.TransactionResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedTransaction)

	assert.Equal(suite.T(), "", updatedTransaction.Data.Note)
}

func (suite *TestSuiteStandard) TestUpdateTransactionSourceDestinationEqual() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Note: "More tests something something", Amount: decimal.NewFromFloat(1253.17)})

	r := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, map[string]any{
		"destinationAccountId": transaction.Data.SourceAccountID,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteStandard) TestUpdateTransactionBrokenJSON() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(5883.53)})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, `{ "amount": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateTransactionInvalidType() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(5883.53)})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, map[string]any{
		"amount": false,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateTransactionInvalidBudgetID() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(5883.53)})

	// Sets the BudgetID to uuid.Nil
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, models.TransactionCreate{})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateTransactionNegativeAmount() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(382.18)})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, `{ "amount": -58.23 }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateTransactionEmptySourceDestinationAccount() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(382.18)})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, models.TransactionCreate{SourceAccountID: uuid.New()})
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)

	recorder = test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, models.TransactionCreate{DestinationAccountID: uuid.New()})
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateNoEnvelopeTransactionOutgoing() {
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{})

	c := models.TransactionCreate{
		BudgetID:             suite.createTestBudget(suite.T(), models.BudgetCreate{Name: "Testing budget for updating of outgoing transfer"}).Data.ID,
		SourceAccountID:      suite.createTestAccount(suite.T(), models.AccountCreate{Name: "Internal Source Account", External: false}).Data.ID,
		DestinationAccountID: suite.createTestAccount(suite.T(), models.AccountCreate{Name: "External destination account", External: true}).Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Amount:               decimal.NewFromFloat(984.13),
	}

	transaction := suite.createTestTransaction(suite.T(), c)

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, `{ "envelopeId": null }`)
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateEnvelopeTransactionOutgoing() {
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{})

	c := models.TransactionCreate{
		BudgetID:             suite.createTestBudget(suite.T(), models.BudgetCreate{Name: "Testing budget for updating of outgoing transfer"}).Data.ID,
		SourceAccountID:      suite.createTestAccount(suite.T(), models.AccountCreate{Name: "Internal Source Account", External: false}).Data.ID,
		DestinationAccountID: suite.createTestAccount(suite.T(), models.AccountCreate{Name: "External destination account", External: true}).Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Amount:               decimal.NewFromFloat(984.13),
	}

	transaction := suite.createTestTransaction(suite.T(), c)
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, fmt.Sprintf("{ \"envelopeId\": \"%s\" }", &envelope.Data.ID))
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateNonExistingEnvelopeTransactionOutgoing() {
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{})

	c := models.TransactionCreate{
		BudgetID:             suite.createTestBudget(suite.T(), models.BudgetCreate{Name: "Testing budget for updating of outgoing transfer"}).Data.ID,
		SourceAccountID:      suite.createTestAccount(suite.T(), models.AccountCreate{Name: "Internal Source Account", External: false}).Data.ID,
		DestinationAccountID: suite.createTestAccount(suite.T(), models.AccountCreate{Name: "External destination account", External: true}).Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Amount:               decimal.NewFromFloat(984.13),
	}

	transaction := suite.createTestTransaction(suite.T(), c)
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, transaction.Data.Links.Self, `{ "envelopeId": "e6fa8eb5-5f2c-4292-8ef9-02f0c2af1ce4" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateNonExistingTransaction() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/transactions/6ae3312c-23cf-4225-9a81-4f218ba41b00", `{ "note": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestDeleteTransaction() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(123.12)})

	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, transaction.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteStandard) TestDeleteNonExistingTransaction() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/transactions/4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestDeleteTransactionWithBody() {
	transaction := suite.createTestTransaction(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(17.21)})
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, transaction.Data.Links.Self, `{ "amount": "23.91" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteStandard) TestDeleteNullTransaction() {
	r := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/transactions/00000000-0000-0000-0000-000000000000", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}
