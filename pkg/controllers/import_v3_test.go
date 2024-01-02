package controllers_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/controllers"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) parseCsvV3(t *testing.T, accountID uuid.UUID, file string) controllers.ImportPreviewListV3 {
	path := fmt.Sprintf("ynab-import-preview?accountId=%s", accountID.String())

	// Parse the test CSV
	body, headers := suite.loadTestFile(fmt.Sprintf("importer/ynab-import/%s", file))
	recorder := test.Request(suite.controller, t, http.MethodPost, fmt.Sprintf("http://example.com/v3/import/%s", path), body, headers)
	assertHTTPStatus(t, &recorder, http.StatusOK)

	// Decode the response
	var response controllers.ImportPreviewListV3
	suite.decodeResponse(&recorder, &response)

	return response
}

// TestImportV3Success verifies successful imports for all import types.
func (suite *TestSuiteStandard) TestImportV3Success() {
	accountID := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "TestImport"}).Data.ID.String()

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
			recorder := test.Request(suite.controller, t, http.MethodPost, fmt.Sprintf("http://example.com/v3/import/%s", tt.path), body, headers)
			assertHTTPStatus(t, &recorder, tt.status)
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
	body, headers := suite.loadTestFile("importer/Budget.yfull")
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v3/import/ynab4?budgetName=Test Budget", body, headers)
	assertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponseV3
	suite.decodeResponse(&recorder, &budget)

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
			recorder := test.Request(suite.controller, t, http.MethodGet, strings.Replace(budget.Data.Links.Month, "YYYY-MM", tt.month.String(), 1), "")
			assertHTTPStatus(t, &recorder, http.StatusOK)
			var month controllers.MonthResponseV3
			suite.decodeResponse(&recorder, &month)

			assert.True(t, decimal.NewFromFloat32(tt.available).Equal(month.Data.Available), "Available for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.available), month.Data.Available)
			assert.True(t, decimal.NewFromFloat32(tt.balance).Equal(month.Data.Balance), "Balance for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.balance), month.Data.Balance)
			assert.True(t, decimal.NewFromFloat32(tt.spent).Equal(month.Data.Spent), "Spent for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.spent), month.Data.Spent)
			assert.True(t, decimal.NewFromFloat32(tt.income).Equal(month.Data.Income), "Income for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.income), month.Data.Income)
		})
	}
}

// TestImportYnab4V3Fails tests failing imports for the YNAB 4 budget import endpoint.
func (suite *TestSuiteStandard) TestImportYnab4V3Fails() {
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
			_ = suite.createTestBudgetV3(suite.T(), models.BudgetCreate{Name: "Import Test"})
		}},
		{"Database error. This test must be the last one.", "Nope. DB is closed.", "there is a problem with the database connection", http.StatusInternalServerError, "", func() {
			suite.CloseDB()
		}},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			tt.preTest()

			path := fmt.Sprintf("http://example.com/v3/import/ynab4?budgetName=%s", tt.budgetName)

			var body *bytes.Buffer
			var headers map[string]string
			var recorder httptest.ResponseRecorder
			if tt.file != "" {
				body, headers = suite.loadTestFile(tt.file)
				recorder = test.Request(suite.controller, t, http.MethodPost, path, body, headers)
			} else {
				recorder = test.Request(suite.controller, t, http.MethodPost, path, "")
			}

			assertHTTPStatus(t, &recorder, tt.status)
			assert.Contains(t, test.DecodeError(t, recorder.Body.Bytes()), tt.expectedError)
		})
	}
}

// TestImportYnabImportPreviewV3Fails tests failing requests for the YNAB import format preview endpoint.
func (suite *TestSuiteStandard) TestImportYnabImportPreviewV3Fails() {
	accountID := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "TestImportYnabImportPreviewV3Fails"}).Data.ID.String()

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
			path := fmt.Sprintf("http://example.com/v3/import/ynab-import-preview?accountId=%s", tt.accountID)

			var body *bytes.Buffer
			var headers map[string]string
			var recorder httptest.ResponseRecorder
			if tt.file != "" {
				body, headers = suite.loadTestFile(tt.file)
				recorder = test.Request(suite.controller, t, http.MethodPost, path, body, headers)
			} else {
				recorder = test.Request(suite.controller, t, http.MethodPost, path, "")
			}

			assertHTTPStatus(t, &recorder, tt.status)
			assert.Contains(t, test.DecodeError(t, recorder.Body.Bytes()), tt.expectedError)
		})
	}
}

func (suite *TestSuiteStandard) TestImportYnabImportPreviewV3DuplicateDetection() {
	// Create test account
	account := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "TestImportYnabImportPreviewV3DuplicateDetection"})

	// Get the import hash of the first transaction and create one with the same import hash
	preview := suite.parseCsvV3(suite.T(), account.Data.ID, "comdirect-ynap.csv")

	transaction := suite.createTestTransactionV3(suite.T(), models.Transaction{
		SourceAccountID: account.Data.ID,
		ImportHash:      preview.Data[0].Transaction.ImportHash,
		Amount:          decimal.NewFromFloat(1.13),
	})

	_ = suite.createTestTransactionV3(suite.T(), models.Transaction{
		SourceAccountID: suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Note: "This account is in a different Budget, but has the same ImportHash", Name: "TestYnabImportPreviewDuplicateDetection Different Budget"}).Data.ID,
		ImportHash:      preview.Data[0].Transaction.ImportHash,
		Amount:          decimal.NewFromFloat(42.23),
	})

	preview = suite.parseCsvV3(suite.T(), account.Data.ID, "comdirect-ynap.csv")

	suite.Assert().Len(preview.Data[0].DuplicateTransactionIDs, 1, "Duplicate transaction IDs field does not have the correct number of IDs")
	suite.Assert().Equal(transaction.Data.ID, preview.Data[0].DuplicateTransactionIDs[0], "Duplicate transaction ID is not ID of the transaction that is duplicated")
}

func (suite *TestSuiteStandard) TestImportYnabImportPreviewV3AvailableFrom() {
	// Create test account
	account := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "TestImportYnabImportPreviewV3AvailableFrom"})
	preview := suite.parseCsvV3(suite.T(), account.Data.ID, "available-from-test.csv")

	dates := []types.Month{
		types.NewMonth(2019, 2),
		types.NewMonth(2019, 4),
		types.NewMonth(2019, 5),
	}

	for i, transaction := range preview.Data {
		assert.Equal(suite.T(), dates[i], transaction.Transaction.AvailableFrom)
	}
}

func (suite *TestSuiteStandard) TestImportYnabImportPreviewV3FindAccounts() {
	// Create a budget and two existing accounts to use
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	edeka := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: budget.Data.ID, Name: "Edeka", External: true})

	// Create an account named "Edeka" in another budget to ensure it is not found. If it were found, the tests for the non-archived
	// Edeka account being found would fail since we do not use an account if we find more than one with the same name
	_ = suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "Edeka"})

	// Account we import to
	internalAccount := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: budget.Data.ID, Name: "Envelope Zero Account"})

	// Test envelope and  test transaction to the Edeka account with an envelope to test the envelope prefill
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: budget.Data.ID}).Data.ID})
	envelopeID := envelope.Data.ID
	_ = suite.createTestTransactionV3(suite.T(), models.Transaction{BudgetID: budget.Data.ID, SourceAccountID: internalAccount.Data.ID, DestinationAccountID: edeka.Data.ID, EnvelopeID: &envelopeID, Amount: decimal.NewFromFloat(12.00)})

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
			preview := suite.parseCsvV3(t, internalAccount.Data.ID, "account-find-test.csv")

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

func (suite *TestSuiteStandard) TestImportYnabImportPreviewV3Match() {
	// Create a budget and two existing accounts to use
	budget := suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	edeka := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: budget.Data.ID, Name: "Edeka", External: true})
	bahn := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: budget.Data.ID, Name: "Deutsche Bahn", External: true})

	// Account we import to
	internalAccount := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{BudgetID: budget.Data.ID, Name: "Envelope Zero Account"})

	// Test envelope and  test transaction to the Edeka account with an envelope to test the envelope prefill
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{CategoryID: suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{BudgetID: budget.Data.ID}).Data.ID})
	envelopeID := envelope.Data.ID
	_ = suite.createTestTransactionV3(suite.T(), models.Transaction{BudgetID: budget.Data.ID, SourceAccountID: internalAccount.Data.ID, DestinationAccountID: edeka.Data.ID, EnvelopeID: &envelopeID, Amount: decimal.NewFromFloat(12.00)})

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
				edeka := suite.createTestMatchRuleV3(t, models.MatchRuleCreate{
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
				edeka := suite.createTestMatchRuleV3(t, models.MatchRuleCreate{
					Match:     "EDEKA*",
					AccountID: edeka.Data.ID,
				})

				db := suite.createTestMatchRuleV3(t, models.MatchRuleCreate{
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
			preview := suite.parseCsvV3(t, internalAccount.Data.ID, "match-rule-test.csv")

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
					suite.controller.DB.Delete(&models.MatchRule{}, id)
				}
			}
		})
	}
}

// TestImportV3Get verifies that the links for the /v3/import path are set correctly.
func (suite *TestSuiteStandard) TestImportV3Get() {
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v3/import", "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)

	var links controllers.ImportV3Response
	suite.decodeResponse(&r, &links)

	assert.Equal(suite.T(), controllers.ImportV3Response{
		Links: controllers.ImportV3Links{
			Ynab4:             "http://example.com/v3/import/ynab4",
			YnabImportPreview: "http://example.com/v3/import/ynab-import-preview",
		},
	}, links)
}
