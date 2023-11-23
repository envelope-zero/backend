package controllers_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestYnab4ImportFails tests failing imports for the YNAB 4 budget import endpoint.
func (suite *TestSuiteStandard) TestYnab4ImportFails() {
	tests := []struct {
		name          string
		budgetName    string
		expectedError string
		status        int
		file          string
		preTest       func()
	}{
		{"No budget name", "", "The budgetName parameter must be set", http.StatusBadRequest, "", func() {}},
		{"No file sent", "same", "you must send a file to this endpoint", http.StatusBadRequest, "", func() {}},
		{"Wrong file name", "same", "this endpoint only supports .yfull files", http.StatusBadRequest, "importer/wrong-name.json", func() {}},
		{"Empty file", "same", "not a valid YNAB4 Budget.yfull file: unexpected end of JSON input", http.StatusBadRequest, "importer/EmptyFile.yfull", func() {}},
		{"Duplicate budget name", "Import Test", "This budget name is already in use", http.StatusBadRequest, "", func() {
			_ = suite.createTestBudget(models.BudgetCreate{Name: "Import Test"})
		}},
		{"Database error. This test must be the last one.", "Nope. DB is closed.", "There is a problem with the database connection", http.StatusInternalServerError, "", func() {
			suite.CloseDB()
		}},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			tt.preTest()

			path := fmt.Sprintf("http://example.com/v1/import/ynab4?budgetName=%s", tt.budgetName)

			var body *bytes.Buffer
			var headers map[string]string
			var recorder httptest.ResponseRecorder
			if tt.file != "" {
				body, headers = suite.loadTestFile(tt.file)
				recorder = test.Request(suite.controller, suite.T(), http.MethodPost, path, body, headers)
			} else {
				recorder = test.Request(suite.controller, suite.T(), http.MethodPost, path, "")
			}

			assert.Equal(t, tt.status, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
			assert.Contains(t, test.DecodeError(t, recorder.Body.Bytes()), tt.expectedError)
		})
	}
}

// TestYnabImportPreviewFails tests failing requests for the YNAB import format preview endpoint.
func (suite *TestSuiteStandard) TestYnabImportPreviewFails() {
	accountID := suite.createTestAccount(models.AccountCreate{Name: "TestYnabImportPreviewFails"}).Data.ID.String()

	tests := []struct {
		name          string
		accountID     string
		status        int
		expectedError string
		file          string
	}{
		{"No account ID", "", http.StatusBadRequest, "the accountId parameter must be set", ""},
		{"Broken ID", "NotAUUID", http.StatusBadRequest, "the specified resource ID is not a valid UUID", "importer/ynab-import/empty.csv"},
		{"No account with ID", "d2525c4f-2f45-49ba-9c5d-75d6b1c26f56", http.StatusNotFound, "there is no Account with this ID", "importer/ynab-import/empty.csv"},
		{"No file sent", accountID, http.StatusBadRequest, "you must send a file to this endpoint", ""},
		{"Wrong file name", accountID, http.StatusBadRequest, "this endpoint only supports .csv files", "importer/ynab-import/wrong-suffix.json"},
		{"Broken upload", accountID, http.StatusBadRequest, "error in line 4 of the CSV: could not parse time: parsing time \"03.23.2020\" as \"01/02/2006\": cannot parse \".23.2020\" as \"/\"", "importer/ynab-import/error-date.csv"},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("http://example.com/v1/import/ynab-import-preview?accountId=%s", tt.accountID)

			var body *bytes.Buffer
			var headers map[string]string
			var recorder httptest.ResponseRecorder
			if tt.file != "" {
				body, headers = suite.loadTestFile(tt.file)
				recorder = test.Request(suite.controller, suite.T(), http.MethodPost, path, body, headers)
			} else {
				recorder = test.Request(suite.controller, suite.T(), http.MethodPost, path, "")
			}

			assert.Equal(t, tt.status, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
			assert.Contains(t, test.DecodeError(t, recorder.Body.Bytes()), tt.expectedError)
		})
	}
}

func (suite *TestSuiteStandard) TestImport() {
	accountID := suite.createTestAccount(models.AccountCreate{Name: "TestImport"}).Data.ID.String()

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
			body, headers := suite.loadTestFile(tt.file)
			recorder := test.Request(suite.controller, suite.T(), http.MethodPost, fmt.Sprintf("http://example.com/v1/import/%s", tt.path), body, headers)
			suite.Assert().Equal(tt.status, recorder.Code, "Request ID %s, response %s", recorder.Header().Get("x-request-id"), recorder.Body.String())
		})
	}
}

func (suite *TestSuiteStandard) TestYnabImportPreviewDuplicateDetection() {
	// Create test account
	account := suite.createTestAccount(models.AccountCreate{Name: "TestYnabImportPreviewDuplicateDetection"})

	// Get the import hash of the first transaction and create one with the same import hash
	preview := parseCSV(suite, account.Data.ID, "comdirect-ynap.csv")

	transaction := suite.createTestTransaction(models.TransactionCreate{
		SourceAccountID: account.Data.ID,
		ImportHash:      preview.Data[0].Transaction.ImportHash,
		Amount:          decimal.NewFromFloat(1.13),
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		SourceAccountID: suite.createTestAccount(models.AccountCreate{Note: "This account is in a different Budget, but has the same ImportHash", Name: "TestYnabImportPreviewDuplicateDetection Different Budget"}).Data.ID,
		ImportHash:      preview.Data[0].Transaction.ImportHash,
		Amount:          decimal.NewFromFloat(42.23),
	})

	preview = parseCSV(suite, account.Data.ID, "comdirect-ynap.csv")

	suite.Assert().Len(preview.Data[0].DuplicateTransactionIDs, 1, "Duplicate transaction IDs field does not have the correct number of IDs")
	suite.Assert().Equal(transaction.Data.ID, preview.Data[0].DuplicateTransactionIDs[0], "Duplicate transaction ID is not ID of the transaction that is duplicated")
}

func (suite *TestSuiteStandard) TestYnabImportAvailableFrom() {
	// Create test account
	account := suite.createTestAccount(models.AccountCreate{Name: "TestYnabImportAvailableFrom"})
	preview := parseCSV(suite, account.Data.ID, "available-from-test.csv")

	dates := []types.Month{
		types.NewMonth(2019, 2),
		types.NewMonth(2019, 4),
		types.NewMonth(2019, 5),
	}

	for i, transaction := range preview.Data {
		assert.Equal(suite.T(), dates[i], transaction.Transaction.AvailableFrom)
	}
}

func parseCSV(suite *TestSuiteStandard, accountID uuid.UUID, file string) controllers.ImportPreviewList {
	path := fmt.Sprintf("ynab-import-preview?accountId=%s", accountID.String())

	// Parse the test CSV
	body, headers := suite.loadTestFile(fmt.Sprintf("importer/ynab-import/%s", file))
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, fmt.Sprintf("http://example.com/v1/import/%s", path), body, headers)
	suite.Assert().Equal(http.StatusOK, recorder.Code, "Request ID %s, response %s", recorder.Header().Get("x-request-id"), recorder.Body.String())

	// Decode the response
	var response controllers.ImportPreviewList
	suite.decodeResponse(&recorder, &response)

	return response
}

func (suite *TestSuiteStandard) TestYnabImportFindAccounts() {
	// Create a budget and two existing accounts to use
	budget := suite.createTestBudget(models.BudgetCreate{})
	edeka := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID, Name: "Edeka", External: true})

	// Create an account named "Edeka" in another budget to ensure it is not found. If it were found, the tests for the non-archived
	// Edeka account being found would fail since we do not use an account if we find more than one with the same name
	_ = suite.createTestAccount(models.AccountCreate{Name: "Edeka"})

	// Account we import to
	internalAccount := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID, Name: "Envelope Zero Account"})

	// Test envelope and  test transaction to the Edeka account with an envelope to test the envelope prefill
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID}).Data.ID})
	envelopeID := envelope.Data.ID
	_ = suite.createTestTransaction(models.TransactionCreate{BudgetID: budget.Data.ID, SourceAccountID: internalAccount.Data.ID, DestinationAccountID: edeka.Data.ID, EnvelopeID: &envelopeID, Amount: decimal.NewFromFloat(12.00)})

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
			preview := parseCSV(suite, internalAccount.Data.ID, "account-find-test.csv")

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

func (suite *TestSuiteStandard) TestMatch() {
	// Create a budget and two existing accounts to use
	budget := suite.createTestBudget(models.BudgetCreate{})
	edeka := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID, Name: "Edeka", External: true})
	bahn := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID, Name: "Deutsche Bahn", External: true})

	// Account we import to
	internalAccount := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID, Name: "Envelope Zero Account"})

	// Test envelope and  test transaction to the Edeka account with an envelope to test the envelope prefill
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID}).Data.ID})
	envelopeID := envelope.Data.ID
	_ = suite.createTestTransaction(models.TransactionCreate{BudgetID: budget.Data.ID, SourceAccountID: internalAccount.Data.ID, DestinationAccountID: edeka.Data.ID, EnvelopeID: &envelopeID, Amount: decimal.NewFromFloat(12.00)})

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
				edeka := suite.createTestMatchRule(t, models.MatchRuleCreate{
					Match:     "EDEKA*",
					AccountID: edeka.Data.ID,
				})

				return [3]uuid.UUID{edeka.ID}
			},
		},
		{
			"Rule for Edeka and DB",
			[]uuid.UUID{internalAccount.Data.ID, internalAccount.Data.ID, uuid.Nil},
			[]uuid.UUID{edeka.Data.ID, bahn.Data.ID, internalAccount.Data.ID},
			[]*uuid.UUID{&envelopeID, nil, nil},
			func(t *testing.T) [3]uuid.UUID {
				edeka := suite.createTestMatchRule(t, models.MatchRuleCreate{
					Match:     "EDEKA*",
					AccountID: edeka.Data.ID,
				})

				db := suite.createTestMatchRule(t, models.MatchRuleCreate{
					Match:     "DB Vertrieb GmbH",
					AccountID: bahn.Data.ID,
				})

				return [3]uuid.UUID{edeka.ID, db.ID}
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			matchRuleIDs := tt.preTest(t)
			preview := parseCSV(suite, internalAccount.Data.ID, "match-rule-test.csv")

			for i, transaction := range preview.Data {
				line := i + 1
				if tt.sourceAccountIDs[i] != uuid.Nil {
					assert.Equal(t, tt.sourceAccountIDs[i], transaction.Transaction.SourceAccountID, "sourceAccountID does not match in line %d", line)
				}

				if tt.destinationAccountIDs[i] != uuid.Nil {
					assert.Equal(t, tt.destinationAccountIDs[i], transaction.Transaction.DestinationAccountID, "destinationAccountID does not match in line %d", line)
				}

				assert.Equal(t, matchRuleIDs[i], transaction.MatchRuleID, "Expected match rule has match '%s', actual match rule has match '%s'", matchRuleIDs[i], transaction.MatchRuleID)

				// This is kept for backwards compatibility and will be removed with API version 3
				// https://github.com/envelope-zero/backend/issues/763
				assert.Equal(t, matchRuleIDs[i], transaction.RenameRuleID, "Expected rename rule has match '%s', actual rename rule has match '%s'", matchRuleIDs[i], transaction.MatchRuleID)

				assert.Equal(t, tt.envelopeIDs[i], transaction.Transaction.EnvelopeID, "proposed envelope ID does not match in line %d", line)
			}

			// Delete match rules
			for _, id := range matchRuleIDs {
				if id != uuid.Nil {
					suite.controller.DB.Delete(&models.MatchRule{}, id)
				}
			}
		})
	}
}
