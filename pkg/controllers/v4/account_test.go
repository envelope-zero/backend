package v4_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	v4 "github.com/envelope-zero/backend/v5/pkg/controllers/v4"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestAccount(t *testing.T, account v4.AccountEditable, expectedStatus ...int) v4.AccountResponse {
	if account.BudgetID == uuid.Nil {
		account.BudgetID = createTestBudget(t, v4.BudgetEditable{Name: "Testing budget"}).Data.ID
	}

	body := []v4.AccountEditable{
		account,
	}

	// Default to 201 Created as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	r := test.Request(t, http.MethodPost, "http://example.com/v4/accounts", body)
	test.AssertHTTPStatus(t, &r, expectedStatus...)

	var a v4.AccountCreateResponse
	test.DecodeResponse(t, &r, &a)

	if r.Code == http.StatusCreated {
		return a.Data[0]
	}

	return v4.AccountResponse{}
}

// TestAccountsDBClosed verifies that errors are processed correctly when
// the database is closed.
func (suite *TestSuiteStandard) TestAccountsDBClosed() {
	b := createTestBudget(suite.T(), v4.BudgetEditable{})

	tests := []struct {
		name string             // Name of the test
		test func(t *testing.T) // Code to run
	}{
		{
			"Creation fails",
			func(t *testing.T) {
				createTestAccount(t, v4.AccountEditable{BudgetID: b.Data.ID}, http.StatusInternalServerError)
			},
		},
		{
			"GET fails",
			func(t *testing.T) {
				recorder := test.Request(t, http.MethodGet, "http://example.com/v4/accounts", "")
				test.AssertHTTPStatus(t, &recorder, http.StatusInternalServerError)

				var response v4.AccountListResponse
				test.DecodeResponse(t, &recorder, &response)
				assert.Contains(t, *response.Error, models.ErrGeneral.Error())
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			suite.CloseDB()

			tt.test(t)
		})
	}
}

// TestAccountOptions verifies that OPTIONS requests are handled correctly.
func (suite *TestSuiteStandard) TestAccountsOptions() {
	tests := []struct {
		name   string
		id     string // path at the Accounts endpoint to test
		status int    // Expected HTTP status code
	}{
		{"No account with this ID", uuid.New().String(), http.StatusNotFound},
		{"Not a valid UUID", "NotParseableAsUUID", http.StatusBadRequest},
		{"Account exists", createTestAccount(suite.T(), v4.AccountEditable{}).Data.ID.String(), http.StatusNoContent},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", "http://example.com/v4/accounts", tt.id)
			r := test.Request(t, http.MethodOptions, path, "")
			test.AssertHTTPStatus(t, &r, tt.status)

			if tt.status == http.StatusNoContent {
				assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", r.Header().Get("allow"))
			}
		})
	}
}

// TestAccountGetSingle verifies that requests for the resource endpoints are
// handled correctly.
func (suite *TestSuiteStandard) TestAccountsGetSingle() {
	a := createTestAccount(suite.T(), v4.AccountEditable{})

	tests := []struct {
		name   string
		id     string
		status int
		method string
	}{
		{"GET Existing account", a.Data.ID.String(), http.StatusOK, http.MethodGet},
		{"GET No account with this ID", uuid.New().String(), http.StatusNotFound, http.MethodGet},
		{"GET Invalid ID (negative number)", "-56", http.StatusBadRequest, http.MethodGet},
		{"GET Invalid ID (positive number)", "23", http.StatusBadRequest, http.MethodGet},
		{"GET Invalid ID (string)", "notaUUID", http.StatusBadRequest, http.MethodGet},
		{"PATCH Invalid ID (negative number)", "-56", http.StatusBadRequest, http.MethodPatch},
		{"PATCH Invalid ID (positive number)", "23", http.StatusBadRequest, http.MethodPatch},
		{"PATCH Invalid ID (string)", "notaUUID", http.StatusBadRequest, http.MethodPatch},
		{"DELETE Invalid ID (negative number)", "-56", http.StatusBadRequest, http.MethodDelete},
		{"DELETE Invalid ID (positive number)", "23", http.StatusBadRequest, http.MethodDelete},
		{"DELETE Invalid ID (string)", "notaUUID", http.StatusBadRequest, http.MethodDelete},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, tt.method, fmt.Sprintf("http://example.com/v4/accounts/%s", tt.id), "")

			var account v4.AccountResponse
			test.DecodeResponse(t, &r, &account)
			test.AssertHTTPStatus(t, &r, tt.status)
		})
	}
}

func (suite *TestSuiteStandard) TestAccountsGetFilter() {
	b1 := createTestBudget(suite.T(), v4.BudgetEditable{})
	b2 := createTestBudget(suite.T(), v4.BudgetEditable{})

	_ = createTestAccount(suite.T(), v4.AccountEditable{
		Name:     "Exact Account Match",
		Note:     "This is a specific note",
		BudgetID: b1.Data.ID,
		OnBudget: true,
		External: false,
	})

	_ = createTestAccount(suite.T(), v4.AccountEditable{
		Name:     "External Account Filter",
		Note:     "This is a specific note",
		BudgetID: b2.Data.ID,
		OnBudget: true,
		External: true,
	})

	_ = createTestAccount(suite.T(), v4.AccountEditable{
		Name:     "External Account Filter",
		Note:     "A different note",
		BudgetID: b1.Data.ID,
		OnBudget: false,
		External: true,
		Archived: true,
	})

	_ = createTestAccount(suite.T(), v4.AccountEditable{
		Name:     "",
		Note:     "specific note",
		BudgetID: b1.Data.ID,
	})

	_ = createTestAccount(suite.T(), v4.AccountEditable{
		Name:     "Name only",
		Note:     "",
		BudgetID: b1.Data.ID,
		Archived: true,
	})

	tests := []struct {
		name      string
		query     string
		len       int
		checkFunc func(t *testing.T, accounts []v4.Account)
	}{
		{"Name single", "name=Exact Account Match", 1, nil},
		{"Name multiple", "name=External Account Filter", 2, nil},
		{"Fuzzy name", "name=Account", 3, nil},
		{"Note", "note=A different note", 1, nil},
		{"Fuzzy Note", "note=note", 4, nil},
		{"Empty name with note", "name=&note=specific", 1, nil},
		{"Empty note with name", "note=&name=Name", 1, nil},
		{"Empty note and name", "note=&name=&onBudget=false", 0, nil},
		{"Budget", fmt.Sprintf("budget=%s", b1.Data.ID), 4, nil},
		{"On budget", "onBudget=true", 1, nil},
		{"Off budget", "onBudget=false", 4, nil},
		{"External", "external=true", 2, nil},
		{"Internal", "external=false", 3, nil},
		{"Not Archived", "archived=false", 3, func(t *testing.T, accounts []v4.Account) {
			for _, a := range accounts {
				assert.False(t, a.Archived)
			}
		}},
		{"Archived", "archived=true", 2, func(t *testing.T, accounts []v4.Account) {
			for _, a := range accounts {
				assert.True(t, a.Archived)
			}
		}},
		{"Search for 'na", "search=na", 3, nil},
		{"Search for 'fi", "search=fi", 4, nil},
		{"Offset 2", "offset=2", 3, nil},
		{"Offset 2, limit 2", "offset=2&limit=2", 2, nil},
		{"Limit 4", "limit=4", 4, nil},
		{"Limit 0", "limit=0", 0, nil},
		{"Limit -1", "limit=-1", 5, nil},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re v4.AccountListResponse
			r := test.Request(t, http.MethodGet, fmt.Sprintf("/v4/accounts?%s", tt.query), "")
			test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
			test.DecodeResponse(t, &r, &re)

			var accountNames []string
			for _, d := range re.Data {
				accountNames = append(accountNames, d.Name)
			}
			assert.Equal(t, tt.len, len(re.Data), "Existing accounts: %#v, Request-ID: %s", strings.Join(accountNames, ", "), r.Header().Get("x-request-id"))

			// Run the custom checks
			if tt.checkFunc != nil {
				tt.checkFunc(t, re.Data)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestAccountsGetMonth() {
	budget := createTestBudget(suite.T(), v4.BudgetEditable{})

	initialBalanceDate := time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)

	sourceAccount := createTestAccount(suite.T(), v4.AccountEditable{
		Name:               "Source Account",
		BudgetID:           budget.Data.ID,
		OnBudget:           true,
		External:           false,
		InitialBalance:     decimal.NewFromFloat(50),
		InitialBalanceDate: &initialBalanceDate,
	})

	destinationAccount := createTestAccount(suite.T(), v4.AccountEditable{
		Name:     "Destination Account",
		BudgetID: budget.Data.ID,
		External: true,
	})

	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{})
	envelopeID := &envelope.Data.ID

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		Date:                 time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10),
		EnvelopeID:           envelopeID,
		SourceAccountID:      sourceAccount.Data.ID,
		DestinationAccountID: destinationAccount.Data.ID,
		ReconciledSource:     true,
	})

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		Date:                 time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10),
		EnvelopeID:           envelopeID,
		SourceAccountID:      sourceAccount.Data.ID,
		DestinationAccountID: destinationAccount.Data.ID,
		ReconciledSource:     false,
	})

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		Date:                  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:                decimal.NewFromFloat(10),
		EnvelopeID:            envelopeID,
		SourceAccountID:       sourceAccount.Data.ID,
		DestinationAccountID:  destinationAccount.Data.ID,
		ReconciledSource:      true,
		ReconciledDestination: true,
	})

	// All tests request the source account
	tests := []struct {
		name                         string
		time                         time.Time
		sourceBalance                float64
		sourceReconciledBalance      float64
		destinationBalance           float64
		destinationReconciledBalance float64
	}{
		{"Before Initial Balance", time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC), 0, 0, 0, 0},
		{"Only Initial Balance", time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC), 50, 50, 0, 0},
		{"After first transaction", time.Date(2023, 10, 20, 0, 0, 0, 0, time.UTC), 40, 40, 10, 0},
		{"After second transaction", time.Date(2023, 11, 20, 0, 0, 0, 0, time.UTC), 30, 40, 20, 0},
		{"After third transaction", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 20, 30, 30, 0}, // destinationReconciledBalance is 0 since external accounts cannot be reconciled
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			recorder := test.Request(t, http.MethodPost, "/v4/accounts/computed", map[string]any{
				"time": tt.time.Format(time.RFC3339),
				"ids":  []string{sourceAccount.Data.ID.String(), destinationAccount.Data.ID.String()},
			},
			)
			test.AssertHTTPStatus(t, &recorder, http.StatusOK)

			var response v4.AccountComputedDataResponse
			test.DecodeResponse(t, &recorder, &response)

			assert.True(t, response.Data[0].Balance.Equal(decimal.NewFromFloat(tt.sourceBalance)), "Source Balance is not correct, expected %f, got %s", tt.sourceBalance, response.Data[0].Balance)
			assert.True(t, response.Data[0].ReconciledBalance.Equal(decimal.NewFromFloat(tt.sourceReconciledBalance)), "Source Reconciled Balance is not correct, expected %f, got %s", tt.sourceReconciledBalance, response.Data[0].ReconciledBalance)

			assert.True(t, response.Data[1].Balance.Equal(decimal.NewFromFloat(tt.destinationBalance)), "Destination Balance is not correct, expected %f, got %s", tt.destinationBalance, response.Data[1].Balance)
			assert.True(t, response.Data[1].ReconciledBalance.Equal(decimal.NewFromFloat(tt.destinationReconciledBalance)), "Destination Reconciled Balance is not correct, expected %f, got %s", tt.destinationReconciledBalance, response.Data[1].ReconciledBalance)
		})
	}
}

func (suite *TestSuiteStandard) TestAccountsCreateFails() {
	// Test account for uniqueness
	a := createTestAccount(suite.T(), v4.AccountEditable{
		Name: "Unique Account Name for Budget",
	})

	tests := []struct {
		name     string
		body     any
		status   int                                            // expected HTTP status
		testFunc func(t *testing.T, a v4.AccountCreateResponse) // tests to perform against the updated account resource
	}{
		{"Broken Body", `[{ "note": 2 }]`, http.StatusBadRequest, func(t *testing.T, a v4.AccountCreateResponse) {
			assert.Equal(t, "json: cannot unmarshal number into Go struct field AccountEditable.note of type string", *a.Error)
		}},
		{
			"No body", "", http.StatusBadRequest,
			func(t *testing.T, a v4.AccountCreateResponse) {
				assert.Equal(t, "the request body must not be empty", *a.Error)
			},
		},
		{
			"No Budget",
			`[{ "note": "Some text" }]`,
			http.StatusNotFound,
			func(t *testing.T, a v4.AccountCreateResponse) {
				assert.Equal(t, "there is no budget matching your query", *a.Data[0].Error)
			},
		},
		{
			"Non-existing Budget",
			`[{ "budgetId": "ea85ad1a-3679-4ced-b83b-89566c12ece9" }]`,
			http.StatusNotFound,
			func(t *testing.T, a v4.AccountCreateResponse) {
				assert.Equal(t, "there is no budget matching your query", *a.Data[0].Error)
			},
		},
		{
			"Duplicate name for budget",
			[]v4.AccountEditable{
				{
					Name:     a.Data.Name,
					BudgetID: a.Data.BudgetID,
				},
			},
			http.StatusBadRequest,
			func(t *testing.T, a v4.AccountCreateResponse) {
				assert.Equal(t, "the account name must be unique for the budget", *a.Data[0].Error)
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPost, "http://example.com/v4/accounts", tt.body)
			test.AssertHTTPStatus(t, &r, tt.status)

			var a v4.AccountCreateResponse
			test.DecodeResponse(t, &r, &a)

			if tt.testFunc != nil {
				tt.testFunc(t, a)
			}
		})
	}
}

// Verify that updating accounts works as desired
func (suite *TestSuiteStandard) TestAccountsUpdate() {
	budget := createTestBudget(suite.T(), v4.BudgetEditable{})
	account := createTestAccount(suite.T(), v4.AccountEditable{Name: "Original name", BudgetID: budget.Data.ID})

	tests := []struct {
		name     string                                   // name of the test
		account  map[string]any                           // the updates to perform. This is not a struct because that would set all fields on the request
		testFunc func(t *testing.T, a v4.AccountResponse) // tests to perform against the updated account resource
	}{
		{
			"Name, On Budget, Note",
			map[string]any{
				"name":     "Another name",
				"onBudget": true,
				"note":     "New note!",
			},
			func(t *testing.T, a v4.AccountResponse) {
				assert.True(t, a.Data.OnBudget)
				assert.Equal(t, "New note!", a.Data.Note)
				assert.Equal(t, "Another name", a.Data.Name)
			},
		},
		{
			"Archived, External",
			map[string]any{
				"archived": true,
				"external": true,
			},
			func(t *testing.T, a v4.AccountResponse) {
				assert.True(t, a.Data.Archived)
				assert.True(t, a.Data.External)
			},
		},
		{
			"Initial Balance",
			map[string]any{
				"initialBalance": "203.21",
			},
			func(t *testing.T, a v4.AccountResponse) {
				assert.True(t, a.Data.InitialBalance.Equal(decimal.NewFromFloat(203.21)))
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(t, http.MethodPatch, account.Data.Links.Self, tt.account)
			test.AssertHTTPStatus(t, &r, http.StatusOK)

			var a v4.AccountResponse
			test.DecodeResponse(t, &r, &a)

			if tt.testFunc != nil {
				tt.testFunc(t, a)
			}
		})
	}
}

func (suite *TestSuiteStandard) TestAccountsUpdateFails() {
	tests := []struct {
		name   string
		id     string
		body   any
		status int // expected response status
	}{
		{"Invalid type", "", `{"name": 2}`, http.StatusBadRequest},
		{"Broken JSON", "", `{ "name": 2" }`, http.StatusBadRequest},
		{"Non-existing account", uuid.New().String(), `{"name": 2}`, http.StatusNotFound},
		{"Set budget to uuid.Nil", "", `{ "budgetId": "00000000-0000-0000-0000-000000000000" }`, http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				account := createTestAccount(suite.T(), v4.AccountEditable{
					Name: "New Budget",
					Note: "More tests something something",
				})

				tt.id = account.Data.ID.String()
			}

			// Update Account
			recorder = test.Request(t, http.MethodPatch, fmt.Sprintf("http://example.com/v4/accounts/%s", tt.id), tt.body)
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestAccountsDelete verifies all cases for Account deletions.
func (suite *TestSuiteStandard) TestAccountsDelete() {
	tests := []struct {
		name   string
		id     string
		status int // expected response status
	}{
		{"Success", "", http.StatusNoContent},
		{"Non-existing Account", uuid.New().String(), http.StatusNotFound},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var recorder httptest.ResponseRecorder

			if tt.id == "" {
				// Create test Account
				a := createTestAccount(t, v4.AccountEditable{})
				tt.id = a.Data.ID.String()
			}

			// Delete Account
			recorder = test.Request(t, http.MethodDelete, fmt.Sprintf("http://example.com/v4/accounts/%s", tt.id), "")
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestAccountsGetSorted verifies that Accounts are sorted by name.
func (suite *TestSuiteStandard) TestAccountsGetSorted() {
	a1 := createTestAccount(suite.T(), v4.AccountEditable{
		Name: "Alphabetically first",
	})

	a2 := createTestAccount(suite.T(), v4.AccountEditable{
		Name: "Second in creation, third in list",
	})

	a3 := createTestAccount(suite.T(), v4.AccountEditable{
		Name: "First is alphabetically second",
	})

	a4 := createTestAccount(suite.T(), v4.AccountEditable{
		Name: "Zulu is the last one",
	})

	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v4/accounts", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)

	var accounts v4.AccountListResponse
	test.DecodeResponse(suite.T(), &r, &accounts)

	require.Len(suite.T(), accounts.Data, 4, "Account list has wrong length")

	assert.Equal(suite.T(), a1.Data.Name, accounts.Data[0].Name)
	assert.Equal(suite.T(), a2.Data.Name, accounts.Data[2].Name)
	assert.Equal(suite.T(), a3.Data.Name, accounts.Data[1].Name)
	assert.Equal(suite.T(), a4.Data.Name, accounts.Data[3].Name)
}

func (suite *TestSuiteStandard) TestAccountsPagination() {
	for i := 0; i < 10; i++ {
		createTestAccount(suite.T(), v4.AccountEditable{Name: fmt.Sprint(i)})
	}

	tests := []struct {
		name          string
		offset        uint
		limit         int
		expectedCount int
		expectedTotal int64
	}{
		{"All", 0, -1, 10, 10},
		{"First 5", 0, 5, 5, 10},
		{"Last 5", 5, -1, 5, 10},
		{"Offset 3", 3, -1, 7, 10},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v4/accounts?offset=%d&limit=%d", tt.offset, tt.limit), "")
			test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)

			var accounts v4.AccountListResponse
			test.DecodeResponse(t, &r, &accounts)

			assert.Equal(suite.T(), tt.offset, accounts.Pagination.Offset)
			assert.Equal(suite.T(), tt.limit, accounts.Pagination.Limit)
			assert.Equal(suite.T(), tt.expectedCount, accounts.Pagination.Count)
			assert.Equal(suite.T(), tt.expectedTotal, accounts.Pagination.Total)
		})
	}
}

func (suite *TestSuiteStandard) TestAccountRecentEnvelopes() {
	budget := createTestBudget(suite.T(), v4.BudgetEditable{})

	account := createTestAccount(suite.T(), v4.AccountEditable{
		BudgetID:       budget.Data.ID,
		Name:           "Internal Account",
		OnBudget:       true,
		External:       false,
		InitialBalance: decimal.NewFromFloat(170),
	})

	externalAccount := createTestAccount(suite.T(), v4.AccountEditable{
		BudgetID: budget.Data.ID,
		Name:     "External Account",
		External: true,
	})

	category := createTestCategory(suite.T(), v4.CategoryEditable{
		BudgetID: budget.Data.ID,
	})

	envelopeIDs := []*uuid.UUID{}
	for i := 0; i < 3; i++ {
		archived := false
		if i%2 == 0 {
			archived = true
		}

		envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{
			CategoryID: category.Data.ID,
			Name:       strconv.Itoa(i),
			Archived:   archived,
		})

		envelopeIDs = append(envelopeIDs, &envelope.Data.ID)

		// Sleep for 10 milliseconds because we only save timestamps with 1 millisecond accuracy
		// This is needed because the test runs so fast that all envelopes are sometimes created
		// within the same millisecond, making the result non-deterministic
		time.Sleep(1 * time.Millisecond)
	}

	// Create 15 transactions:
	//  * 2 for the first envelope
	//  * 2 for the second envelope
	//  * 11 for the last envelope
	for i := 0; i < 15; i++ {
		eIndex := i
		if i > 5 {
			eIndex = 2
		}
		_ = createTestTransaction(suite.T(), v4.TransactionEditable{
			EnvelopeID:           envelopeIDs[eIndex%3],
			SourceAccountID:      externalAccount.Data.ID,
			DestinationAccountID: account.Data.ID,
			Amount:               decimal.NewFromFloat(17.45),
		})
	}

	// Create three income transactions
	//
	// This is a regression test for income always showing at the last
	// position in the recent envelopes (before the LIMIT) since count(id) for
	// income was always 0. This is due to the envelope ID for income being NULL
	// and count() not counting NULL values.
	//
	// Creating three income transactions puts "income" as the second most common
	// envelope, verifying the fix
	for i := 0; i < 3; i++ {
		_ = createTestTransaction(suite.T(), v4.TransactionEditable{
			EnvelopeID:           nil,
			SourceAccountID:      externalAccount.Data.ID,
			DestinationAccountID: account.Data.ID,
			Amount:               decimal.NewFromFloat(1337.42),
		})
	}

	r := test.Request(suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v4/accounts/%s/recent-envelopes", account.Data.ID), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)

	var recentEnvelopeResponse v4.RecentEnvelopesResponse
	test.DecodeResponse(suite.T(), &r, &recentEnvelopeResponse)

	data := recentEnvelopeResponse.Data
	suite.Require().Len(data, 4, "The number of envelopes in recentEnvelopes is not correct, expected 4, got %d", len(data))

	// The last envelope needs to be the first in the sort since it
	// has been the most common one
	suite.Assert().Equal(envelopeIDs[2], data[0].ID)
	suite.Assert().Equal(true, data[0].Archived)

	// Income is the second one since it appears three times
	var nilUUIDPointer *uuid.UUID
	suite.Assert().Equal(nilUUIDPointer, data[1].ID)
	suite.Assert().Equal(false, data[1].Archived)

	// Order for envelopes with the same frequency is undefined and therefore not tested
	// Only one of the two is archived, but since the order is undefined we XOR them
	suite.Assert().Equal(true, data[2].Archived != data[3].Archived)
}
