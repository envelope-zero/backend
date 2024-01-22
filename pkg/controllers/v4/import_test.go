package v4_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/envelope-zero/backend/v5/internal/types"
	v4 "github.com/envelope-zero/backend/v5/pkg/controllers/v4"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) parseCSV(t *testing.T, accountID uuid.UUID, file string) v4.ImportPreviewList {
	path := fmt.Sprintf("ynab-import-preview?accountId=%s", accountID.String())

	// Parse the test CSV
	body, headers := test.LoadTestFile(t, fmt.Sprintf("importer/ynab-import/%s", file))
	recorder := test.Request(t, http.MethodPost, fmt.Sprintf("http://example.com/v4/import/%s", path), body, headers)
	test.AssertHTTPStatus(t, &recorder, http.StatusOK)

	// Decode the response
	var response v4.ImportPreviewList
	test.DecodeResponse(t, &recorder, &response)

	return response
}

// TestImportSuccess verifies successful imports for all import types.
func (suite *TestSuiteStandard) TestImportSuccess() {
	accountID := createTestAccount(suite.T(), v4.AccountEditable{Name: "TestImport"}).Data.ID.String()

	tests := []struct {
		name   string
		path   string
		file   string
		status int
	}{
		{"Import whole budget", "ynab4?budgetName=Test Budget", "importer/Budget.yfull", http.StatusCreated},
		{"Preview transaction import", fmt.Sprintf("ynab-import-preview?accountId=%s", accountID), "importer/ynab-import/comdirect-ynap.csv", http.StatusOK},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			// Import one
			body, headers := test.LoadTestFile(t, tt.file)
			recorder := test.Request(t, http.MethodPost, fmt.Sprintf("http://example.com/v4/import/%s", tt.path), body, headers)
			test.AssertHTTPStatus(t, &recorder, tt.status)
		})
	}
}

// TestImportYnab4BudgetCalculation verifies that the budget calculation is correct
// for an imported budget from YNAB 4.
//
// This in turn tests the budget calculation itself for edge cases that can only happen
// with YNAB 4 budgets, e.g. for overspend handling migration
//
// Resource creation for YNAB 4 imports is tested in pkg/importer/parser/ynab4/parse_test.go -> TestParse,
// but budget calculation is a controller function and therefore is tested here
func (suite *TestSuiteStandard) TestImportYnab4BudgetCalculation() {
	body, headers := test.LoadTestFile(suite.T(), "importer/Budget.yfull")
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v4/import/ynab4?budgetName=Test Budget", body, headers)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget v4.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	// In YNAB 4, starting balance counts as income our outflow, in Envelope Zero it does not
	// Therefore, the numbers for available, balance, spent and income will differ in some cases
	tests := []struct {
		month     types.Month
		available float32
		balance   float32
		spent     float32
		budgeted  float32
		income    float32
	}{
		{types.NewMonth(2022, 10), 46.17, -100, -175, 75, 0},
		{types.NewMonth(2022, 11), 906.17, -60, -100, 140, 1000},
		{types.NewMonth(2022, 12), 886.17, -55, -110, 115, 95},
		{types.NewMonth(2023, 1), 576.17, 55, 0, 0, 0},
		{types.NewMonth(2023, 2), 456.17, 175, 0, 0, 0},
	}

	for _, tt := range tests {
		suite.T().Run(tt.month.String(), func(t *testing.T) {
			// Get the budget caculations for
			recorder := test.Request(t, http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", tt.month.String(), 1), "")
			test.AssertHTTPStatus(t, &recorder, http.StatusOK)
			var month v4.MonthResponse
			test.DecodeResponse(t, &recorder, &month)

			assert.True(t, decimal.NewFromFloat32(tt.available).Equal(month.Data.Available), "Available for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.available), month.Data.Available)
			assert.True(t, decimal.NewFromFloat32(tt.balance).Equal(month.Data.Balance), "Balance for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.balance), month.Data.Balance)
			assert.True(t, decimal.NewFromFloat32(tt.spent).Equal(month.Data.Spent), "Spent for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.spent), month.Data.Spent)
			assert.True(t, decimal.NewFromFloat32(tt.income).Equal(month.Data.Income), "Income for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.income), month.Data.Income)
		})
	}
}

// TestImportYnab4Fails tests failing imports for the YNAB 4 budget import endpoint.
func (suite *TestSuiteStandard) TestImportYnab4Fails() {
	tests := []struct {
		name          string
		budgetName    string
		expectedError string
		status        int
		file          string
		preTest       func()
	}{
		{"No budget name", "", "the budgetName parameter must be set", http.StatusBadRequest, "", func() {}},
		{"No file sent", "same", "you must send a file to this endpoint", http.StatusBadRequest, "", func() {}},
		{"Wrong file name", "same", "this endpoint only supports files of the following types: .yfull", http.StatusBadRequest, "importer/wrong-name.json", func() {}},
		{"Empty file", "same", "not a valid YNAB4 Budget.yfull file: unexpected end of JSON input", http.StatusBadRequest, "importer/EmptyFile.yfull", func() {}},
		{"Duplicate budget name", "Import Test", "this budget name is already in use", http.StatusBadRequest, "", func() {
			_ = createTestBudget(suite.T(), v4.BudgetEditable{Name: "Import Test"})
		}},
		{"Database error. This test must be the last one.", "Nope. DB is closed.", models.ErrGeneral.Error(), http.StatusInternalServerError, "", func() {
			suite.CloseDB()
		}},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			tt.preTest()

			path := fmt.Sprintf("http://example.com/v4/import/ynab4?budgetName=%s", tt.budgetName)

			var body *bytes.Buffer
			var headers map[string]string
			var recorder httptest.ResponseRecorder
			if tt.file != "" {
				body, headers = test.LoadTestFile(t, tt.file)
				recorder = test.Request(t, http.MethodPost, path, body, headers)
			} else {
				recorder = test.Request(t, http.MethodPost, path, "")
			}

			test.AssertHTTPStatus(t, &recorder, tt.status)
			var response v4.BudgetResponse
			test.DecodeResponse(t, &recorder, &response)
			assert.Contains(t, *response.Error, tt.expectedError)
		})
	}
}

// TestImportYnabImportPreviewFails tests failing requests for the YNAB import format preview endpoint.
func (suite *TestSuiteStandard) TestImportYnabImportPreviewFails() {
	accountID := createTestAccount(suite.T(), v4.AccountEditable{Name: "TestImportYnabImportPreviewFails"}).Data.ID.String()

	tests := []struct {
		name          string
		accountID     string
		status        int
		expectedError string
		file          string
	}{
		{"No account ID", "", http.StatusBadRequest, "the accountId parameter must be set", ""},
		{"Broken ID", "NotAUUID", http.StatusBadRequest, "the specified resource ID is not a valid UUID", "importer/ynab-import/empty.csv"},
		{"No account with ID", "d2525c4f-2f45-49ba-9c5d-75d6b1c26f56", http.StatusNotFound, "there is no account matching your query", "importer/ynab-import/empty.csv"},
		{"No file sent", accountID, http.StatusBadRequest, "you must send a file to this endpoint", ""},
		{"Wrong file name", accountID, http.StatusBadRequest, "this endpoint only supports files of the following types: .csv", "importer/ynab-import/wrong-suffix.json"},
		{"Broken upload", accountID, http.StatusBadRequest, "error in line 4 of the CSV: could not parse time: parsing time \"03.23.2020\" as \"01/02/2006\": cannot parse \".23.2020\" as \"/\"", "importer/ynab-import/error-date.csv"},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("http://example.com/v4/import/ynab-import-preview?accountId=%s", tt.accountID)

			var body *bytes.Buffer
			var headers map[string]string
			var recorder httptest.ResponseRecorder
			if tt.file != "" {
				body, headers = test.LoadTestFile(t, tt.file)
				recorder = test.Request(t, http.MethodPost, path, body, headers)
			} else {
				recorder = test.Request(t, http.MethodPost, path, "")
			}

			test.AssertHTTPStatus(t, &recorder, tt.status)
			var response v4.ImportPreviewList
			test.DecodeResponse(t, &recorder, &response)
			assert.Equal(t, tt.expectedError, *response.Error)
		})
	}
}

func (suite *TestSuiteStandard) TestImportYnabImportPreviewDuplicateDetection() {
	// Create test account
	account := createTestAccount(suite.T(), v4.AccountEditable{Name: "TestImportYnabImportPreviewDuplicateDetection"})

	// Get the import hash of the first transaction and create one with the same import hash
	preview := suite.parseCSV(suite.T(), account.Data.ID, "comdirect-ynap.csv")

	transaction := createTestTransaction(suite.T(), v4.TransactionEditable{
		SourceAccountID: account.Data.ID,
		ImportHash:      preview.Data[0].Transaction.ImportHash,
		Amount:          decimal.NewFromFloat(1.13),
	})

	_ = createTestTransaction(suite.T(), v4.TransactionEditable{
		SourceAccountID: createTestAccount(suite.T(), v4.AccountEditable{Note: "This account is in a different Budget, but has the same ImportHash", Name: "TestYnabImportPreviewDuplicateDetection Different Budget"}).Data.ID,
		ImportHash:      preview.Data[0].Transaction.ImportHash,
		Amount:          decimal.NewFromFloat(42.23),
	})

	preview = suite.parseCSV(suite.T(), account.Data.ID, "comdirect-ynap.csv")

	suite.Assert().Len(preview.Data[0].DuplicateTransactionIDs, 1, "Duplicate transaction IDs field does not have the correct number of IDs")
	suite.Assert().Equal(transaction.Data.ID, preview.Data[0].DuplicateTransactionIDs[0], "Duplicate transaction ID is not ID of the transaction that is duplicated")
}

func (suite *TestSuiteStandard) TestImportYnabImportPreviewAvailableFrom() {
	// Create test account
	account := createTestAccount(suite.T(), v4.AccountEditable{Name: "TestImportYnabImportPreviewAvailableFrom"})
	preview := suite.parseCSV(suite.T(), account.Data.ID, "available-from-test.csv")

	dates := []types.Month{
		types.NewMonth(2019, 2),
		types.NewMonth(2019, 4),
		types.NewMonth(2019, 5),
	}

	for i, transaction := range preview.Data {
		assert.Equal(suite.T(), dates[i], transaction.Transaction.AvailableFrom)
	}
}

func (suite *TestSuiteStandard) TestImportYnabImportPreviewFindAccounts() {
	// Create a budget and two existing accounts to use
	budget := createTestBudget(suite.T(), v4.BudgetEditable{})
	edeka := createTestAccount(suite.T(), v4.AccountEditable{BudgetID: budget.Data.ID, Name: "Edeka", External: true})

	// Create an account named "Edeka" in another budget to ensure it is not found. If it were found, the tests for the non-archived
	// Edeka account being found would fail since we do not use an account if we find more than one with the same name
	_ = createTestAccount(suite.T(), v4.AccountEditable{Name: "Edeka"})

	// Account we import to
	internalAccount := createTestAccount(suite.T(), v4.AccountEditable{BudgetID: budget.Data.ID, Name: "Envelope Zero Account"})

	// Test envelope and  test transaction to the Edeka account with an envelope to test the envelope prefill
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{CategoryID: createTestCategory(suite.T(), v4.CategoryEditable{BudgetID: budget.Data.ID}).Data.ID})
	envelopeID := envelope.Data.ID
	_ = createTestTransaction(suite.T(), v4.TransactionEditable{SourceAccountID: internalAccount.Data.ID, DestinationAccountID: edeka.Data.ID, EnvelopeID: &envelopeID, Amount: decimal.NewFromFloat(12.00)})

	tests := []struct {
		name                    string       // Name of the test
		sourceAccountIDs        []uuid.UUID  // The IDs of the source accounts
		sourceAccountNames      []string     // The sourceAccountName attribute after the find has been performed
		destinationAccountIDs   []uuid.UUID  // The IDs of the destination accounts
		destinationAccountNames []string     // The destinationAccountName attribute after the find has been performed
		envelopeIDs             []*uuid.UUID // expected IDs of envelopes
	}{
		{
			"No matching (Some Company) & 1 Matching (Edeka) accounts",
			[]uuid.UUID{internalAccount.Data.ID, internalAccount.Data.ID, uuid.Nil},
			[]string{"", "", "Some Company"},
			[]uuid.UUID{edeka.Data.ID, uuid.Nil, internalAccount.Data.ID},
			[]string{"Edeka", "Deutsche Bahn", ""},
			[]*uuid.UUID{&envelopeID, nil, nil},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			preview := suite.parseCSV(t, internalAccount.Data.ID, "account-find-test.csv")

			for i, transaction := range preview.Data {
				// Add 2 since the loop is 0-indexed but the CSV data begins at row 2 (line 1 is the header row)
				line := i + 2

				assert.Equal(t, tt.sourceAccountNames[i], transaction.SourceAccountName, "sourceAccountName does not match in line %d", line)
				assert.Equal(t, tt.destinationAccountNames[i], transaction.DestinationAccountName, "destinationAccountName does not match in line %d", line)

				assert.Equal(t, tt.envelopeIDs[i], transaction.Transaction.EnvelopeID, "proposed envelope ID does not match in line %d", line)

				if tt.sourceAccountIDs[i] != uuid.Nil {
					assert.Equal(t, tt.sourceAccountIDs[i], transaction.Transaction.SourceAccountID, "sourceAccountID does not match in line %d", line)
				}

				if tt.destinationAccountIDs[i] != uuid.Nil {
					assert.Equal(t, tt.destinationAccountIDs[i], transaction.Transaction.DestinationAccountID, "destinationAccountID does not match in line %d", line)
				}
			}
		})
	}
}

func (suite *TestSuiteStandard) TestImportYnabImportPreviewMatch() {
	// Create a budget and two existing accounts to use
	budget := createTestBudget(suite.T(), v4.BudgetEditable{})
	edeka := createTestAccount(suite.T(), v4.AccountEditable{BudgetID: budget.Data.ID, Name: "Edeka", External: true})
	bahn := createTestAccount(suite.T(), v4.AccountEditable{BudgetID: budget.Data.ID, Name: "Deutsche Bahn", External: true})

	// Account we import to
	internalAccount := createTestAccount(suite.T(), v4.AccountEditable{BudgetID: budget.Data.ID, Name: "Envelope Zero Account"})

	// Test envelope and  test transaction to the Edeka account with an envelope to test the envelope prefill
	envelope := createTestEnvelope(suite.T(), v4.EnvelopeEditable{CategoryID: createTestCategory(suite.T(), v4.CategoryEditable{BudgetID: budget.Data.ID}).Data.ID})
	envelopeID := envelope.Data.ID
	_ = createTestTransaction(suite.T(), v4.TransactionEditable{SourceAccountID: internalAccount.Data.ID, DestinationAccountID: edeka.Data.ID, EnvelopeID: &envelopeID, Amount: decimal.NewFromFloat(12.00)})

	tests := []struct {
		name                  string                        // Name of the test
		sourceAccountIDs      []uuid.UUID                   // The IDs of the source accounts
		destinationAccountIDs []uuid.UUID                   // The IDs of the destination accounts
		envelopeIDs           []*uuid.UUID                  // expected IDs of envelopes
		preTest               func(*testing.T) [3]uuid.UUID // Function to execute before running tests
	}{
		{
			"Rule for Edeka",
			[]uuid.UUID{internalAccount.Data.ID, internalAccount.Data.ID, uuid.Nil},
			[]uuid.UUID{edeka.Data.ID, uuid.Nil, internalAccount.Data.ID},
			[]*uuid.UUID{&envelopeID, nil, nil},
			func(t *testing.T) [3]uuid.UUID {
				edeka := createTestMatchRule(t, v4.MatchRuleEditable{
					Match:     "EDEKA*",
					AccountID: edeka.Data.ID,
				})

				return [3]uuid.UUID{edeka.Data.ID}
			},
		},
		{
			"Rule for Edeka and DB",
			[]uuid.UUID{internalAccount.Data.ID, internalAccount.Data.ID, uuid.Nil},
			[]uuid.UUID{edeka.Data.ID, bahn.Data.ID, internalAccount.Data.ID},
			[]*uuid.UUID{&envelopeID, nil, nil},
			func(t *testing.T) [3]uuid.UUID {
				edeka := createTestMatchRule(t, v4.MatchRuleEditable{
					Match:     "EDEKA*",
					AccountID: edeka.Data.ID,
				})

				db := createTestMatchRule(t, v4.MatchRuleEditable{
					Match:     "DB Vertrieb GmbH",
					AccountID: bahn.Data.ID,
				})

				return [3]uuid.UUID{edeka.Data.ID, db.Data.ID}
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			matchRuleIDs := tt.preTest(t)
			preview := suite.parseCSV(t, internalAccount.Data.ID, "match-rule-test.csv")

			for i, transaction := range preview.Data {
				line := i + 1
				if tt.sourceAccountIDs[i] != uuid.Nil {
					assert.Equal(t, tt.sourceAccountIDs[i], transaction.Transaction.SourceAccountID, "sourceAccountID does not match in line %d", line)
				}

				if tt.destinationAccountIDs[i] != uuid.Nil {
					assert.Equal(t, tt.destinationAccountIDs[i], transaction.Transaction.DestinationAccountID, "destinationAccountID does not match in line %d", line)
				}

				if matchRuleIDs[i] != uuid.Nil {
					assert.Equal(t, matchRuleIDs[i], *transaction.MatchRuleID, "Expected match rule has match '%s', actual match rule has match '%s'", matchRuleIDs[i], transaction.MatchRuleID)
				}

				assert.Equal(t, tt.envelopeIDs[i], transaction.Transaction.EnvelopeID, "proposed envelope ID does not match in line %d", line)
			}

			// Delete match rules
			for _, id := range matchRuleIDs {
				if id != uuid.Nil {
					models.DB.Delete(&models.MatchRule{}, id)
				}
			}
		})
	}
}

// TestImportGet verifies that the links for the //import path are set correctly.
func (suite *TestSuiteStandard) TestImportGet() {
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v4/import", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)

	var links v4.ImportResponse
	test.DecodeResponse(suite.T(), &r, &links)

	assert.Equal(suite.T(), v4.ImportResponse{
		Links: v4.ImportLinks{
			Ynab4:             "http://example.com/v4/import/ynab4",
			YnabImportPreview: "http://example.com/v4/import/ynab-import-preview",
		},
	}, links)
}
