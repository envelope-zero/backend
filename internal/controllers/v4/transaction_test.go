package v4_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	v4 "github.com/envelope-zero/backend/v5/internal/controllers/v4"
	"github.com/envelope-zero/backend/v5/internal/httputil"
	"github.com/envelope-zero/backend/v5/internal/models"
	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// createTestTransaction creates a test transactions via the v4 API.
func createTestTransaction(t *testing.T, transaction v4.TransactionEditable, expectedStatus ...int) v4.TransactionResponse {
	if transaction.SourceAccountID == uuid.Nil {
		transaction.SourceAccountID = createTestAccount(t, v4.AccountEditable{Name: "Source Account"}).Data.ID
	}

	if transaction.DestinationAccountID == uuid.Nil {
		transaction.DestinationAccountID = createTestAccount(t, v4.AccountEditable{Name: "Destination Account"}).Data.ID
	}

	if transaction.EnvelopeID == &uuid.Nil {
		*transaction.EnvelopeID = createTestEnvelope(t, v4.EnvelopeEditable{Name: "Transaction Test Envelope"}).Data.ID
	}

	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	reqBody := []v4.TransactionEditable{transaction}

	r := test.Request(t, http.MethodPost, "http://example.com/v4/transactions", reqBody)
	test.AssertHTTPStatus(t, &r, expectedStatus...)

	var tr v4.TransactionCreateResponse
	test.DecodeResponse(t, &r, &tr)

	return tr.Data[0]
}

// TestTransactionsOptions verifies that the HTTP OPTIONS response for //transactions/{id} is correct.
func (suite *TestSuiteStandard) TestTransactionsOptions() {
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
				return createTestTransaction(suite.T(), v4.TransactionEditable{Amount: decimal.NewFromFloat(31)}).Data.Links.Self
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var p string
			if tt.pathFunc != nil {
				p = tt.pathFunc()
			} else {
				p = fmt.Sprintf("%s/%s", "http://example.com/v4/transactions", tt.id)
			}

			r := test.Request(t, http.MethodOptions, p, "")
			test.AssertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestTransactionsDatabaseError verifies that the endpoints return the appropriate
// error when the database is disconncted.
func (suite *TestSuiteStandard) TestTransactionsDatabaseError() {
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

			recorder := test.Request(t, tt.method, fmt.Sprintf("http://example.com/v4/transactions%s", tt.path), tt.body)
			test.AssertHTTPStatus(t, &recorder, http.StatusInternalServerError)

			var response v4.TransactionListResponse
			test.DecodeResponse(t, &recorder, &response)
			assert.Equal(t, models.ErrGeneral.Error(), *response.Error)
		})
	}
}

// TestTransactionsGet verifies that transactions can be read from the API.
// It also acts as a regression test for a bug where transactions were sorted by date(date)
// instead of datetime(date), leading to transactions being correctly sorted by dates, but
// not correctly sorted when multiple transactions occurred on a day. In that case, the
// oldest transaction would be at the bottom and not at the top.
func (suite *TestSuiteStandard) TestTransactionsGet() {
	t1 := createTestTransaction(suite.T(), v4.TransactionEditable{
		Amount: decimal.NewFromFloat(17.23),
		Date:   time.Date(2023, 11, 10, 10, 11, 12, 0, time.UTC),
	})

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		Amount: decimal.NewFromFloat(23.42),
		Date:   time.Date(2023, 11, 10, 11, 12, 13, 0, time.UTC),
	})

	// Need to sleep 1 second because SQLite datetime only has second precision
	time.Sleep(1 * time.Second)

	t3 := createTestTransaction(suite.T(), v4.TransactionEditable{
		Amount: decimal.NewFromFloat(44.05),
		Date:   time.Date(2023, 11, 10, 10, 11, 12, 0, time.UTC),
	})

	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v4/transactions", "")

	var response v4.TransactionListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 3)

	// Verify that the transaction with the earlier date is the last in the list
	assert.Equal(suite.T(), t1.Data.ID, response.Data[2].ID, t1.Data.CreatedAt)

	// Verify that the transaction added for the same time as the first, but added later
	// is before the other
	assert.Equal(suite.T(), t3.Data.ID, response.Data[1].ID, t3.Data.CreatedAt)
}

// TestTransactionsGetFilter verifies that filtering transactions works as expected.
func (suite *TestSuiteStandard) TestTransactionsGetFilter() {
	b := createTestBudget(suite.T(), v4.BudgetEditable{})

	a1 := createTestAccount(suite.T(), v4.AccountEditable{BudgetID: b.Data.ID, Name: "TestTransactionsGetFilter 1", OnBudget: true})
	a2 := createTestAccount(suite.T(), v4.AccountEditable{BudgetID: b.Data.ID, Name: "TestTransactionsGetFilter 2", External: true})
	a3 := createTestAccount(suite.T(), v4.AccountEditable{BudgetID: b.Data.ID, Name: "TestTransactionsGetFilter 3", OnBudget: true})

	c := createTestCategory(suite.T(), v4.CategoryEditable{BudgetID: b.Data.ID})

	e1 := createTestEnvelope(suite.T(), v4.EnvelopeEditable{CategoryID: c.Data.ID})
	e2 := createTestEnvelope(suite.T(), v4.EnvelopeEditable{CategoryID: c.Data.ID})

	e1ID := &e1.Data.ID
	e2ID := &e2.Data.ID

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		Date:                  time.Date(2018, 9, 5, 17, 13, 29, 45256, time.UTC),
		Amount:                decimal.NewFromFloat(2.718),
		Note:                  "This was an important expense",
		EnvelopeID:            e1ID,
		SourceAccountID:       a1.Data.ID,
		DestinationAccountID:  a2.Data.ID,
		ReconciledSource:      true,
		ReconciledDestination: false,
	})

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		Date:                  time.Date(2016, 5, 1, 14, 13, 25, 584575, time.UTC),
		AvailableFrom:         types.NewMonth(2016, 5),
		Amount:                decimal.NewFromFloat(11235.813),
		Note:                  "Not important",
		EnvelopeID:            e2ID,
		SourceAccountID:       a2.Data.ID,
		DestinationAccountID:  a1.Data.ID,
		ReconciledSource:      false,
		ReconciledDestination: true,
	})

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		Date:                  time.Date(2021, 2, 6, 5, 1, 0, 585, time.UTC),
		Amount:                decimal.NewFromFloat(2.718),
		Note:                  "",
		EnvelopeID:            e1ID,
		SourceAccountID:       a3.Data.ID,
		DestinationAccountID:  a2.Data.ID,
		ReconciledSource:      false,
		ReconciledDestination: false,
	})

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		Date:                  time.Date(2024, 11, 24, 0, 0, 0, 0, time.UTC),
		Amount:                decimal.NewFromFloat(1),
		Note:                  "This is a transfer",
		EnvelopeID:            nil,
		SourceAccountID:       a1.Data.ID,
		DestinationAccountID:  a3.Data.ID,
		ReconciledSource:      false,
		ReconciledDestination: false,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"After all dates", fmt.Sprintf("fromDate=%s", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 1},
		{"After date", fmt.Sprintf("fromDate=%s", time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 3},
		{"Amount less or equal to 2.71", "amountLessOrEqual=2.71", 1},
		{"Amount less or equal to 2.718", "amountLessOrEqual=2.718", 3},
		{"Amount less or equal to 1000", "amountLessOrEqual=1000", 3},
		{"Amount more or equal to 1 and less than 3", "amountMoreOrEqual=1&amountLessOrEqual=3", 3},
		{"Amount more or equal to 2.718", "amountMoreOrEqual=2.718", 3},
		{"Amount more or equal to 100 and less than 10", "amountMoreOrEqual=100&amountLessOrEqual=10", 0},
		{"Amount more or equal to 100", "amountMoreOrEqual=100", 1},
		{"Amount more or equal to 11235.813", "amountMoreOrEqual=11235.813", 1},
		{"Amount more or equal to 99999", "amountMoreOrEqual=99999", 0},
		{"Available after - no transactions", fmt.Sprintf("availableFromFromDate=%s", time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 2},       // Available from is only relevant for income, but set for all transactions
		{"Available after - returns transactions", fmt.Sprintf("availableFromFromDate=%s", time.Date(2000, 12, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 4}, // Available from is only relevant for income, but set for all transactions
		{"Available at date - no transactions", fmt.Sprintf("availableFromDate=%s", time.Date(2016, 5, 2, 11, 17, 0, 0, time.UTC).Format(time.RFC3339Nano)), 0},
		{"Available at month - with transaction", fmt.Sprintf("availableFromDate=%s", time.Date(2016, 5, 1, 12, 53, 15, 148041, time.UTC).Format(time.RFC3339Nano)), 1},
		{"Available before - no transactions", fmt.Sprintf("availableFromUntilDate=%s", time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 0},       // Needs to be before 2016-05-01T00:00:00Z since that's what the transaction defaults to when created
		{"Available before - returns transactions", fmt.Sprintf("availableFromUntilDate=%s", time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 4}, // Available from is only relevant for income, but set for all transactions
		{"Before all dates", fmt.Sprintf("untilDate=%s", time.Date(2010, 8, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 0},
		{"Before date", fmt.Sprintf("untilDate=%s", time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 1},
		{"Between two dates", fmt.Sprintf("untilDate=%s&fromDate=%s", time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano), time.Date(2015, 12, 24, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 2},
		{"Budget and Note", fmt.Sprintf("budget=%s&note=Not", b.Data.ID), 1},
		{"Budget Match", fmt.Sprintf("budget=%s", b.Data.ID), 4},
		{"Destination Account", fmt.Sprintf("destination=%s", a2.Data.ID), 2},
		{"Type=INCOME", "type=INCOME", 1},
		{"Type=SPEND", "type=SPEND", 2},
		{"Type=TRANSFER", "type=TRANSFER", 1},
		{"Direction=IN", "direction=IN", 1},
		{"Direction=OUT", "direction=OUT", 2},
		{"Direction=INTERNAL and Budget ID", fmt.Sprintf("budget=%s&direction=INTERNAL", b.Data.ID), 1},
		{"Direction=INTERNAL and TYPE=TRANSFER", "direction=INTERNAL&type=TRANSFER", 1}, // Testing both at the same time to ensure functioning SQL
		{"Envelope 2", fmt.Sprintf("envelope=%s", e2.Data.ID), 1},
		{"Exact Amount", fmt.Sprintf("amount=%s", decimal.NewFromFloat(2.718).String()), 2},
		{"Exact Date", fmt.Sprintf("date=%s", time.Date(2021, 2, 6, 5, 1, 0, 585, time.UTC).Format(time.RFC3339Nano)), 1},
		{"Existing Account 1", fmt.Sprintf("account=%s", a1.Data.ID), 3},
		{"Existing Account 2", fmt.Sprintf("account=%s", a2.Data.ID), 3},
		{"Fuzzy note", "note=important", 2},
		{"Impossible between two dates", fmt.Sprintf("fromDate=%s&untilDate=%s", time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano), time.Date(2015, 12, 24, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 0},
		{"Limit and Fuzzy Note", "limit=1&note=important", 1},
		{"Limit and Offset", "limit=1&offset=1", 1},
		{"Limit negative", "limit=-123", 4},
		{"Limit positive", "limit=2", 2},
		{"Limit unset", "limit=-1", 4},
		{"Limit zero", "limit=0", 0},
		{"No note", "note=", 1},
		{"Non-existing Account", "account=534a3562-c5e8-46d1-a2e2-e96c00e7efec", 0},
		{"Non-existing Source Account", "source=3340a084-acf8-4cb4-8f86-9e7f88a86190", 0},
		{"Not reconciled in destination account", "reconciledDestination=false", 3},
		{"Not reconciled in source account", "reconciledSource=false", 3},
		{"Note", "note=Not important", 1},
		{"Offset and Fuzzy Note", "offset=2&note=important", 0},
		{"Offset higher than number", "offset=5", 0},
		{"Offset positive", "offset=2", 2},
		{"Offset zero", "offset=0", 4},
		{"Reconciled in destination account", "reconciledDestination=true", 1},
		{"Reconciled in source account", "reconciledSource=true", 1},
		{"Regression - For 'account', query needs to be ORed between the accounts and ANDed with all other conditions", fmt.Sprintf("note=&account=%s", a2.Data.ID), 1},
		{"Regression #749", fmt.Sprintf("untilDate=%s", time.Date(2021, 2, 6, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)), 3},
		{"Same date", fmt.Sprintf("date=%s", time.Date(2021, 2, 6, 7, 0, 0, 700, time.UTC).Format(time.RFC3339Nano)), 1},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re v4.TransactionListResponse
			r := test.Request(t, http.MethodGet, fmt.Sprintf("/v4/transactions?%s", tt.query), "")
			test.AssertHTTPStatus(t, &r, http.StatusOK)
			test.DecodeResponse(t, &r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

// TestTransactionsGetInvalidQuery verifies that invalid filtering queries
// return a HTTP Bad Request.
func (suite *TestSuiteStandard) TestTransactionsGetInvalidQuery() {
	tests := []string{
		"source=MaybeADog",
		"destination=OrARat?",
		"envelope=NopeDefinitelyAMole",
		"date=A long time ago",
		"amount=Seventeen Cents",
		"reconciledSource=I don't think so",
		"account=ItIsAHippo!",
		"offset=-1",          // offset is a uint
		"limit=name",         // limit is an int
		"direction=external", // direction needs to be a TransactionDirection, external does not exist
		"type=winnings",      // type needs to be a TransactionType, winnings don't exist (would be nice though, right?)
	}

	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(t, http.MethodGet, fmt.Sprintf("http://example.com/v4/transactions?%s", tt), "")
			test.AssertHTTPStatus(t, &recorder, http.StatusBadRequest)

			var body v4.TransactionListResponse
			test.DecodeResponse(t, &recorder, &body)

			assert.Len(t, body.Data, 0)
			assert.NotEmpty(t, body.Error)
		})
	}
}

// TestTransactionsCreateInvalidBody verifies that creation of transactions
// with an unparseable request body returns a HTTP Bad Request.
func (suite *TestSuiteStandard) TestTransactionsCreateInvalidBody() {
	r := test.Request(suite.T(), http.MethodPost, "http://example.com/v4/transactions", `{ Invalid request": Body }`)
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	var tr v4.TransactionCreateResponse
	test.DecodeResponse(suite.T(), &r, &tr)

	assert.Equal(suite.T(), httputil.ErrInvalidBody.Error(), *tr.Error)
	assert.Nil(suite.T(), tr.Data)
}

// TestTransactionsCreate verifies that transaction creation works.
func (suite *TestSuiteStandard) TestTransactionsCreate() {
	budget := createTestBudget(suite.T(), v4.BudgetEditable{})
	internalAccount := createTestAccount(suite.T(), v4.AccountEditable{External: false, BudgetID: budget.Data.ID, Name: "TestTransactionsCreate Internal"})
	externalAccount := createTestAccount(suite.T(), v4.AccountEditable{External: true, BudgetID: budget.Data.ID, Name: "TestTransactionsCreate External"})

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
					SourceAccountID: uuid.New(),
					Amount:          decimal.NewFromFloat(17.23),
					Note:            "v4 non-existing budget ID",
				},
				{
					SourceAccountID:      internalAccount.Data.ID,
					DestinationAccountID: externalAccount.Data.ID,
					Amount:               decimal.NewFromFloat(57.01),
				},
			},
			http.StatusNotFound,
			nil,
			[]string{
				"invalid source account: there is no account matching your query",
				"",
			},
		},
		{
			"Both succeed",
			[]models.Transaction{
				{
					SourceAccountID:      internalAccount.Data.ID,
					DestinationAccountID: externalAccount.Data.ID,
					Amount:               decimal.NewFromFloat(17.23),
				},
				{
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
			r := test.Request(t, http.MethodPost, "http://example.com/v4/transactions", tt.transactions)
			test.AssertHTTPStatus(t, &r, tt.expectedStatus)

			var tr v4.TransactionCreateResponse
			test.DecodeResponse(t, &r, &tr)

			for i, transaction := range tr.Data {
				if tt.expectedErrors[i] == "" {
					assert.Equal(t, fmt.Sprintf("http://example.com/v4/transactions/%s", transaction.Data.ID), transaction.Data.Links.Self)
				} else {
					// This needs to be in the else to prevent nil pointer errors since we're dereferencing pointers
					assert.Equal(t, tt.expectedErrors[i], *transaction.Error)
				}
			}
		})
	}
}

// TestTransactionsGetSingle verifies that a transaction can be read from the API via its link
// and that the link is for API v4.
func (suite *TestSuiteStandard) TestTransactionsGetSingle() {
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
				return createTestTransaction(suite.T(), v4.TransactionEditable{Amount: decimal.NewFromFloat(13.71)}).Data.Links.Self
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
				p = fmt.Sprintf("%s/%s", "http://example.com/v4/transactions", tt.id)
			}

			r := test.Request(t, http.MethodGet, p, "")
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestTransactionsDelete verifies the correct success and error responses
// for DELETE requests.
func (suite *TestSuiteStandard) TestTransactionsDelete() {
	tests := []struct {
		name   string // Name for the test
		status int    // Expected HTTP status
		id     string // String to use as ID.
	}{
		{
			"Standard deletion",
			http.StatusNoContent,
			createTestTransaction(suite.T(), v4.TransactionEditable{Amount: decimal.NewFromFloat(123.12)}).Data.ID.String(),
		},
		{
			"Does not exist",
			http.StatusNotFound,
			"4bcb6d09-ced1-41e8-a3fe-bf4f16c5e501",
		},
		{
			"Null transaction",
			http.StatusNotFound,
			"00000000-0000-0000-0000-000000000000",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s", "http://example.com/v4/transactions", tt.id)

			r := test.Request(t, http.MethodDelete, p, "")
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestTransactionsUpdateFail verifies that transaction updates fail where they should.
func (suite *TestSuiteStandard) TestTransactionsUpdateFail() {
	transaction := createTestTransaction(suite.T(), v4.TransactionEditable{Amount: decimal.NewFromFloat(584.42), Note: "Test note for transaction"})

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
			"Negative amount",
			http.StatusBadRequest,
			`{ "amount": -58.23 }`,
		},
		{
			"Empty source account",
			http.StatusNotFound,
			map[string]any{
				"sourceAccountId": uuid.New(),
			},
		},
		{
			"Empty destination account",
			http.StatusNotFound,
			map[string]any{
				"destinationAccountId": uuid.New(),
			},
		},
		{
			"Non-existing envelope",
			http.StatusNotFound,
			`{ "envelopeId": "e6fa8eb5-5f2c-4292-8ef9-02f0c2af1ce4" }`,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPatch, transaction.Data.Links.Self, tt.body)
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

// TestUpdateNonExistingTransaction verifies that patching a non-existent transaction returns a 404.
func (suite *TestSuiteStandard) TestUpdateNonExistingTransaction() {
	recorder := test.Request(suite.T(), http.MethodPatch, "http://example.com/v4/transactions/6ae3312c-23cf-4225-9a81-4f218ba41b00", `{ "note": "2" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

// TestTransactionsUpdate verifies that transaction updates are successful.
func (suite *TestSuiteStandard) TestTransactionsUpdate() {
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{})
	transaction := createTestTransaction(suite.T(), v4.TransactionEditable{
		Amount:               decimal.NewFromFloat(23.14),
		Note:                 "Test note for transaction",
		SourceAccountID:      createTestAccount(suite.T(), v4.AccountEditable{Name: "Internal Source Account", External: false}).Data.ID,
		DestinationAccountID: createTestAccount(suite.T(), v4.AccountEditable{Name: "External destination account", External: true}).Data.ID,
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
			r := test.Request(t, http.MethodPatch, transaction.Data.Links.Self, tt.body)
			test.AssertHTTPStatus(t, &r, http.StatusOK)
		})
	}
}
