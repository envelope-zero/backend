package controllers_test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/envelope-zero/backend/v2/pkg/controllers"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/envelope-zero/backend/v2/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) loadTestFile(filePath string) (*bytes.Buffer, map[string]string) {
	path := path.Join("../../testdata", filePath)
	body := new(bytes.Buffer)

	mw := multipart.NewWriter(body)

	file, err := os.Open(path)
	if err != nil {
		suite.Assert().Fail(err.Error())
	}

	w, err := mw.CreateFormFile("file", filePath)
	if err != nil {
		suite.Assert().Fail(err.Error())
	}

	if _, err := io.Copy(w, file); err != nil {
		suite.Assert().Fail(err.Error())
	}

	mw.Close()

	return body, map[string]string{"Content-Type": mw.FormDataContentType()}
}

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
		{"No file sent", "same", "You must send a file to this endpoint", http.StatusBadRequest, "", func() {}},
		{"Wrong file name", "same", "This endpoint only supports .yfull files", http.StatusBadRequest, "importer/wrong-name.json", func() {}},
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
	accountID := suite.createTestAccount(models.AccountCreate{}).Data.ID.String()

	tests := []struct {
		name          string
		accountID     string
		status        int
		expectedError string
		file          string
	}{
		{"No account ID", "", http.StatusBadRequest, "The accountId parameter must be set", ""},
		{"Broken ID", "NotAUUID", http.StatusBadRequest, "The specified resource ID is not a valid UUID", "importer/ynab-import/empty.csv"},
		{"No account with ID", "d2525c4f-2f45-49ba-9c5d-75d6b1c26f56", http.StatusNotFound, "No Account found for the specified ID", "importer/ynab-import/empty.csv"},
		{"No file sent", accountID, http.StatusBadRequest, "You must send a file to this endpoint", ""},
		{"Wrong file name", accountID, http.StatusBadRequest, "This endpoint only supports .csv files", "importer/ynab-import/wrong-suffix.json"},
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
	accountID := suite.createTestAccount(models.AccountCreate{}).Data.ID.String()

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
	account := suite.createTestAccount(models.AccountCreate{})

	// Get the import hash of the first transaction and create one with the same import hash
	preview := parseComdirectTestCSV(suite, account.Data.ID)

	transaction := suite.createTestTransaction(models.TransactionCreate{
		SourceAccountID: account.Data.ID,
		ImportHash:      preview.Data[0].Model.ImportHash,
		Amount:          decimal.NewFromFloat(1.13),
	})

	preview = parseComdirectTestCSV(suite, account.Data.ID)

	suite.Assert().Len(preview.Data[0].DuplicateTransactionIDs, 1, "Duplicate transaction IDs field does not have the correct number of IDs")
	suite.Assert().Equal(transaction.Data.ID, preview.Data[0].DuplicateTransactionIDs[0], "Duplicate transaction ID is not ID of the transaction that is duplicated")
}

func parseComdirectTestCSV(suite *TestSuiteStandard, accountID uuid.UUID) controllers.ImportPreviewList {
	path := fmt.Sprintf("ynab-import-preview?accountId=%s", accountID.String())

	// Parse the test CSV
	body, headers := suite.loadTestFile("importer/ynab-import/comdirect-ynap.csv")
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, fmt.Sprintf("http://example.com/v1/import/%s", path), body, headers)
	suite.Assert().Equal(http.StatusOK, recorder.Code, "Request ID %s, response %s", recorder.Header().Get("x-request-id"), recorder.Body.String())

	// Decode the response
	var response controllers.ImportPreviewList
	suite.decodeResponse(&recorder, &response)

	return response
}
