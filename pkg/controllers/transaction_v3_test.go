package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v4/pkg/controllers"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// createTestTransactionV3 creates a test transactions via the v3 API.
func (suite *TestSuiteStandard) createTestTransactionV3(t *testing.T, transaction models.Transaction, expectedStatus ...int) controllers.TransactionResponseV3 {
	transaction = suite.defaultTransactionCreate(transaction)

	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	reqBody := []models.Transaction{transaction}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/transactions", reqBody)
	assertHTTPStatus(t, &r, expectedStatus...)

	var tr controllers.TransactionCreateResponseV3
	suite.decodeResponse(&r, &tr)

	return tr.Data[0]
}

// TestTransactionsV3Options verifies that the HTTP OPTIONS response for /v3/transactions/{id} is correct.
func (suite *TestSuiteStandard) TestTransactionsV3Options() {
	tests := []struct {
		name     string        // Name for the test
		status   int           // Expected HTTP status
		id       string        // String to use as ID. Ignored when pathFunc is non-nil
		pathFunc func() string // Function returning the path
	}{
		{
			"Does not exist",
			http.StatusNotFound,
			uuid.New().String(),
			nil,
		},
		{
			"Invalid UUID",
			http.StatusBadRequest,
			"NotParseableAsUUID",
			nil,
		},
		{
			"Success",
			http.StatusNoContent,
			"",
			func() string {
				return suite.createTestTransactionV3(suite.T(), models.Transaction{Amount: decimal.NewFromFloat(31)}).Data.Links.Self
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var p string
			if tt.pathFunc != nil {
				p = tt.pathFunc()
			} else {
				p = fmt.Sprintf("%s/%s", "http://example.com/v3/transactions", tt.id)
			}

			r := test.Request(suite.controller, t, http.MethodOptions, p, "")
			assertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestTransactionsV3DatabaseError verifies that the endpoints return the appropriate
// error when the database is disconncted.
func (suite *TestSuiteStandard) TestTransactionsV3DatabaseError() {
	tests := []struct {
		name   string // Name of the test
		path   string // Path to send request to
		method string // HTTP method to use
		body   string // The request body
	}{
		{"GET Collection", "", http.MethodGet, ""},
		// Skipping POST Collection here since we need to check the indivdual transactions for that one
		{"OPTIONS Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodOptions, ""},
		{"GET Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodGet, ""},
		{"PATCH Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodPatch, ""},
		{"DELETE Single", fmt.Sprintf("/%s", uuid.New().String()), http.MethodDelete, ""},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			suite.CloseDB()

			recorder := test.Request(suite.controller, t, tt.method, fmt.Sprintf("http://example.com/v3/transactions%s", tt.path), tt.body)
			assertHTTPStatus(t, &recorder, http.StatusInternalServerError)
			assert.Equal(t, httperrors.ErrDatabaseClosed.Error(), test.DecodeError(t, recorder.Body.Bytes()))
		})
	}
}

// TestTransactionsV3Get verifies that transactions can be read from the API.
// It also acts as a regression test for a bug where transactions were sorted by date(date)
// instead of datetime(date), leading to transactions being correctly sorted by dates, but
// not correctly sorted when multiple transactions occurred on a day. In that case, the
// oldest transaction would be at the bottom and not at the top.
func (suite *TestSuiteStandard) TestTransactionsV3Get() {
	t1 := suite.createTestTransactionV3(suite.T(), models.Transaction{
		Amount: decimal.NewFromFloat(17.23),
		Date:   time.Date(2023, 11, 10, 10, 11, 12, 0, time.UTC),
	})

	_ = suite.createTestTransactionV3(suite.T(), models.Transaction{
		Amount: decimal.NewFromFloat(23.42),
		Date:   time.Date(2023, 11, 10, 11, 12, 13, 0, time.UTC),
	})

	// Need to sleep 1 second because SQLite datetime only has second precision
	time.Sleep(1 * time.Second)

	t3 := suite.createTestTransactionV3(suite.T(), models.Transaction{
		Amount: decimal.NewFromFloat(44.05),
		Date:   time.Date(2023, 11, 10, 10, 11, 12, 0, time.UTC),
	})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v3/transactions", "")

	var response controllers.TransactionListResponseV3
	suite.decodeResponse(&recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 3)

	// Verify that the transaction with the earlier date is the last in the list
	assert.Equal(suite.T(), t1.Data.ID, response.Data[2].ID, t1.Data.CreatedAt)

	// Verify that the transaction added for the same time as the first, but added later
	// is before the other
	assert.Equal(suite.T(), t3.Data.ID, response.Data[1].ID, t3.Data.CreatedAt)
}

// TestTransactionsV3GetFilter verifies that filtering transactions works as expected.
func (suite *TestSuiteStandard) TestTransactionsV3GetFilter() {
	b := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})

	a1 := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: b.Data.ID, Name: "TestTransactionsV3GetFilter 1"})
	a2 := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: b.Data.ID, Name: "TestTransactionsV3GetFilter 2"})
	a3 := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: b.Data.ID, Name: "TestTransactionsV3GetFilter 3"})

	c := suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: b.Data.ID})

	e1 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: c.Data.ID})
	e2 := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: c.Data.ID})

	e1ID := &e1.Data.ID
	e2ID := &e2.Data.ID

	_ = suite.createTestTransactionV3(suite.T(), models.Transaction{
		Date:                  time.Date(2018, 9, 5, 17, 13, 29, 45256, time.UTC),
		Amount:                decimal.NewFromFloat(2.718),
		Note:                  "This was an important expense",
		BudgetID:              b.Data.ID,
		EnvelopeID:            e1ID,
		SourceAccountID:       a1.Data.ID,
		DestinationAccountID:  a2.Data.ID,
		ReconciledSource:      true,
		ReconciledDestination: false,
	})

	_ = suite.createTestTransactionV3(suite.T(), models.Transaction{
		Date:                  time.Date(2016, 5, 1, 14, 13, 25, 584575, time.UTC),
		Amount:                decimal.NewFromFloat(11235.813),
		Note:                  "Not important",
		BudgetID:              b.Data.ID,
		EnvelopeID:            e2ID,
		SourceAccountID:       a2.Data.ID,
		DestinationAccountID:  a1.Data.ID,
		ReconciledSource:      true,
		ReconciledDestination: true,
	})

	_ = suite.createTestTransactionV3(suite.T(), models.Transaction{
		Date:                  time.Date(2021, 2, 6, 5, 1, 0, 585, time.UTC),
		Amount:                decimal.NewFromFloat(2.718),
		Note:                  "",
		BudgetID:              b.Data.ID,
		EnvelopeID:            e1ID,
		SourceAccountID:       a3.Data.ID,
		DestinationAccountID:  a2.Data.ID,
		ReconciledSource:      false,
		ReconciledDestination: true,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Exact Time", fmt.Sprintf("date=%s", time.Date(2021, 2, 6, 5, 1, 0, 585, time.UTC).Format(time.RFC3339Nano)), 1},
		{"Same date", fmt.Sprintf("date=%s", time.Date(2021, 2, 6, 7, 0, 0, 700, time.UTC).Format(time.RFC3339Nano)), 1},
		{"After date", fmt.Sprintf("fromDate=%s", time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 2},
		{"Before date", fmt.Sprintf("untilDate=%s", time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 1},
		{"After all dates", fmt.Sprintf("fromDate=%s", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 0},
		{"Before all dates", fmt.Sprintf("untilDate=%s", time.Date(2010, 8, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 0},
		{"Regression #749", fmt.Sprintf("untilDate=%s", time.Date(2021, 2, 6, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 3},
		{"Between two dates", fmt.Sprintf("untilDate=%s&fromDate=%s", time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano), time.Date(2015, 12, 24, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 2},
		{"Impossible between two dates", fmt.Sprintf("fromDate=%s&untilDate=%s", time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano), time.Date(2015, 12, 24, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 0},
		{"Exact Amount", fmt.Sprintf("amount=%s", decimal.NewFromFloat(2.718).String()), 2},
		{"Note", "note=Not important", 1},
		{"No note", "note=", 1},
		{"Fuzzy note", "note=important", 2},
		{"Budget Match", fmt.Sprintf("budget=%s", b.Data.ID), 3},
		{"Envelope 2", fmt.Sprintf("envelope=%s", e2.Data.ID), 1},
		{"Non-existing Source Account", "source=3340a084-acf8-4cb4-8f86-9e7f88a86190", 0},
		{"Destination Account", fmt.Sprintf("destination=%s", a2.Data.ID), 2},
		{"Not reconciled in source account", "reconciledSource=false", 1},
		{"Not reconciled in destination account", "reconciledDestination=false", 1},
		{"Reconciled in source account", "reconciledSource=true", 2},
		{"Reconciled in destination account", "reconciledDestination=true", 2},
		{"Non-existing Account", "account=534a3562-c5e8-46d1-a2e2-e96c00e7efec", 0},
		{"Existing Account 2", fmt.Sprintf("account=%s", a2.Data.ID), 3},
		{"Existing Account 1", fmt.Sprintf("account=%s", a1.Data.ID), 2},
		{"Amount less or equal to 2.71", "amountLessOrEqual=2.71", 0},
		{"Amount less or equal to 2.718", "amountLessOrEqual=2.718", 2},
		{"Amount less or equal to 1000", "amountLessOrEqual=1000", 2},
		{"Amount more or equal to 2.718", "amountMoreOrEqual=2.718", 3},
		{"Amount more or equal to 11235.813", "amountMoreOrEqual=11235.813", 1},
		{"Amount more or equal to 99999", "amountMoreOrEqual=99999", 0},
		{"Amount more or equal to 100", "amountMoreOrEqual=100", 1},
		{"Amount more or equal to 100 and less than 10", "amountMoreOrEqual=100&amountLessOrEqual=10", 0},
		{"Amount more or equal to 1 and less than 3", "amountMoreOrEqual=1&amountLessOrEqual=3", 2},
		{"Regression - For 'account', query needs to be ORed between the accounts and ANDed with all other conditions", fmt.Sprintf("note=&account=%s", a2.Data.ID), 1},
		{"Limit positive", "limit=2", 2},
		{"Limit zero", "limit=0", 0},
		{"Limit unset", "limit=-1", 3},
		{"Limit negative", "limit=-123", 3},
		{"Offset zero", "offset=0", 3},
		{"Offset positive", "offset=2", 1},
		{"Offset higher than number", "offset=5", 0},
		{"Limit and Offset", "limit=1&offset=1", 1},
		{"Limit and Fuzzy Note", "limit=1&note=important", 1},
		{"Offset and Fuzzy Note", "offset=2&note=important", 0},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.TransactionListResponseV3
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v3/transactions?%s", tt.query), "")
			assertHTTPStatus(t, &r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

// TestTransactionsV3GetInvalidQuery verifies that invalid filtering queries
// return a HTTP Bad Request.
func (suite *TestSuiteStandard) TestTransactionsV3GetInvalidQuery() {
	tests := []string{
		"budget=DefinitelyACat",
		"source=MaybeADog",
		"destination=OrARat?",
		"envelope=NopeDefinitelyAMole",
		"date=A long time ago",
		"amount=Seventeen Cents",
		"reconciledSource=I don't think so",
		"account=ItIsAHippo!",
		"offset=-1",  // offset is a uint
		"limit=name", // limit is an int
	}

	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("http://example.com/v3/transactions?%s", tt), "")
			assertHTTPStatus(t, &recorder, http.StatusBadRequest)
		})
	}
}

// TestTransactionsV3CreateInvalidBody verifies that creation of transactions
// with an unparseable request body returns a HTTP Bad Request.
func (suite *TestSuiteStandard) TestTransactionsV3CreateInvalidBody() {
	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v3/transactions", `{ Invalid request": Body }`)
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	var tr controllers.TransactionCreateResponseV3
	suite.decodeResponse(&r, &tr)

	assert.Equal(suite.T(), httperrors.ErrInvalidBody.Error(), *tr.Error)
	assert.Nil(suite.T(), tr.Data)
}

// TestTransactionsV3Create verifies that transaction creation works.
func (suite *TestSuiteStandard) TestTransactionsV3Create() {
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	internalAccount := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{External: false, BudgetID: budget.Data.ID, Name: "TestTransactionsV3Create Internal"})
	externalAccount := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{External: true, BudgetID: budget.Data.ID, Name: "TestTransactionsV3Create External"})

	tests := []struct {
		name           string
		transactions   []models.Transaction
		expectedStatus int
		expectedError  *error   // Error expected in the response
		expectedErrors []string // Errors expected for the individual transactions
	}{
		{
			"One success, one fail",
			[]models.Transaction{
				{
					BudgetID: uuid.New(),
					Amount:   decimal.NewFromFloat(17.23),
					Note:     "v3 non-existing budget ID",
				},
				{
					BudgetID:             budget.Data.ID,
					SourceAccountID:      internalAccount.Data.ID,
					DestinationAccountID: externalAccount.Data.ID,
					Amount:               decimal.NewFromFloat(57.01),
				},
			},
			http.StatusNotFound,
			nil,
			[]string{
				"there is no Budget with this ID",
				"",
			},
		},
		{
			"Both succeed",
			[]models.Transaction{
				{
					BudgetID:             budget.Data.ID,
					SourceAccountID:      internalAccount.Data.ID,
					DestinationAccountID: externalAccount.Data.ID,
					Amount:               decimal.NewFromFloat(17.23),
				},
				{
					BudgetID:             budget.Data.ID,
					SourceAccountID:      internalAccount.Data.ID,
					DestinationAccountID: externalAccount.Data.ID,
					Amount:               decimal.NewFromFloat(57.01),
				},
			},
			http.StatusCreated,
			nil,
			[]string{
				"",
				"",
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v3/transactions", tt.transactions)
			assertHTTPStatus(t, &r, tt.expectedStatus)

			var tr controllers.TransactionCreateResponseV3
			suite.decodeResponse(&r, &tr)

			for i, transaction := range tr.Data {
				if tt.expectedErrors[i] == "" {
					assert.Equal(t, fmt.Sprintf("http://example.com/v3/transactions/%s", transaction.Data.ID), transaction.Data.Links.Self)
				} else {
					// This needs to be in the else to prevent nil pointer errors since we're dereferencing pointers
					assert.Equal(t, tt.expectedErrors[i], *transaction.Error)
				}
			}
		})
	}
}

// TestTransactionsV3GetSingle verifies that a transaction can be read from the API via its link
// and that the link is for API v3.
func (suite *TestSuiteStandard) TestTransactionsV3GetSingle() {
	tests := []struct {
		name     string        // Name for the test
		status   int           // Expected HTTP status
		id       string        // String to use as ID. Ignored when pathFunc is non-nil
		pathFunc func() string // Function returning the path
	}{
		{
			"Standard transaction",
			http.StatusOK,
			"",
			func() string {
				return suite.createTestTransactionV3(suite.T(), models.Transaction{Amount: decimal.NewFromFloat(13.71)}).Data.Links.Self
			},
		},
		{
			"Invalid UUID",
			http.StatusBadRequest,
			"NotParseableAsUUID",
			nil,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var p string
			if tt.pathFunc != nil {
				p = tt.pathFunc()
			} else {
				p = fmt.Sprintf("%s/%s", "http://example.com/v3/transactions", tt.id)
			}

			r := test.Request(suite.controller, suite.T(), http.MethodGet, p, "")
			assertHTTPStatus(suite.T(), &r, tt.status)
		})
	}
}

// TestTransactionsV3Delete verifies the correct success and error responses
// for DELETE requests.
func (suite *TestSuiteStandard) TestTransactionsV3Delete() {
	tests := []struct {
		name   string // Name for the test
		status int    // Expected HTTP status
		id     string // String to use as ID.
	}{
		{
			"Standard deletion",
			http.StatusNoContent,
			suite.createTestTransactionV3(suite.T(), models.Transaction{Amount: decimal.NewFromFloat(123.12)}).Data.ID.String(),
		},
		{
			"Does not exist",
			http.StatusNotFound,
			"4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501",
		},
		{
			"Null transaction",
			http.StatusBadRequest,
			"00000000-0000-0000-0000-000000000000",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s", "http://example.com/v3/transactions", tt.id)

			r := test.Request(suite.controller, t, http.MethodDelete, p, "")
			assertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestTransactionsV3UpdateFail verifies that transaction updates fail where they should.
func (suite *TestSuiteStandard) TestTransactionsV3UpdateFail() {
	transaction := suite.createTestTransactionV3(suite.T(), models.Transaction{Amount: decimal.NewFromFloat(584.42), Note: "Test note for transaction"})

	tests := []struct {
		name   string // Name for the test
		status int    // Expected HTTP status
		body   any    // Body to send to the PATCH endpoint
	}{
		{
			"Source Equals Destination",
			http.StatusBadRequest,
			map[string]any{
				"destinationAccountId": transaction.Data.SourceAccountID,
			},
		},
		{
			"Invalid body",
			http.StatusBadRequest,
			`{ "amount": 2" }`,
		},
		{
			"Invalid type",
			http.StatusBadRequest,
			map[string]any{
				"amount": false,
			},
		},
		{
			"Invalid budget ID",
			http.StatusBadRequest,
			`{ "budgetId": "00000000-0000-0000-0000-000000000000" }`,
		},
		{
			"Negative amount",
			http.StatusBadRequest,
			`{ "amount": -58.23 }`,
		},
		{
			"Empty source account",
			http.StatusNotFound,
			models.Transaction{SourceAccountID: uuid.New()},
		},
		{
			"Empty destination account",
			http.StatusNotFound,
			models.Transaction{DestinationAccountID: uuid.New()},
		},
		{
			"Non-existing envelope",
			http.StatusNotFound,
			`{ "envelopeId": "e6fa8eb5-5f2c-4292-8ef9-02f0c2af1ce4" }`,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPatch, transaction.Data.Links.Self, tt.body)
			assertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestUpdateNonExistingTransactionV3 verifies that patching a non-existent transaction returns a 404.
func (suite *TestSuiteStandard) TestUpdateNonExistingTransactionV3() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v3/transactions/6ae3312c-23cf-4225-9a81-4f218ba41b00", `{ "note": "2" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

// TestTransactionsV3Update verifies that transaction updates are successful.
func (suite *TestSuiteStandard) TestTransactionsV3Update() {
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})
	transaction := suite.createTestTransactionV3(suite.T(), models.Transaction{
		Amount:               decimal.NewFromFloat(23.14),
		Note:                 "Test note for transaction",
		BudgetID:             suite.createTestBudgetV3(suite.T(), models.BudgetCreate{Name: "Testing budget for updating of outgoing transfer"}).Data.ID,
		SourceAccountID:      suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "Internal Source Account", External: false}).Data.ID,
		DestinationAccountID: suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "External destination account", External: true}).Data.ID,
		EnvelopeID:           &envelope.Data.ID,
	})

	tests := []struct {
		name string // Name for the test
		body any    // Body to send to the PATCH endpoint
	}{
		{
			"Empty note",
			map[string]any{
				"note": "",
			},
		},
		{
			"No Envelope",
			`{ "envelopeId": null }`,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPatch, transaction.Data.Links.Self, tt.body)
			assertHTTPStatus(t, &r, http.StatusOK)
		})
	}
}
