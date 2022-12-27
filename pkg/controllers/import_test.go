package controllers_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"

	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/test"
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

func (suite *TestSuiteStandard) TestOptionsImport() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/import", "")
	suite.Assert().Equal(http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
	suite.Assert().Equal(recorder.Header().Get("allow"), "OPTIONS, POST")

	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/import/ynab4", "")
	suite.Assert().Equal(http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
	suite.Assert().Equal(recorder.Header().Get("allow"), "OPTIONS, POST")
}

func (suite *TestSuiteStandard) TestImportFails() {
	// Budget name not set
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/import", "")
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
	suite.Assert().Contains(test.DecodeError(suite.T(), recorder.Body.Bytes()), "The budgetName parameter must be set")

	// No file sent
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/import?budgetName=same", "")
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
	suite.Assert().Contains(test.DecodeError(suite.T(), recorder.Body.Bytes()), "You must send a file to this endpoint")

	// Wrong file name
	body, headers := suite.loadTestFile("wrong-name.json")
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/import?budgetName=same", body, headers)
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code, "Request ID %s, response %s", recorder.Header().Get("x-request-id"), recorder.Body.String())
	suite.Assert().Contains(test.DecodeError(suite.T(), recorder.Body.Bytes()), "If you tried to upload a YNAB 4 budget, make sure its file name ends with .yfull")

	// Empty file
	body, headers = suite.loadTestFile("EmptyFile.yfull")
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/import?budgetName=same", body, headers)
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code, "Request ID %s, response %s", recorder.Header().Get("x-request-id"), recorder.Body.String())
	suite.Assert().Contains(test.DecodeError(suite.T(), recorder.Body.Bytes()), "not a valid YNAB4 Budget.yfull file: unexpected end of JSON input")

	// Budget with name already exists
	_ = suite.createTestBudget(models.BudgetCreate{Name: "Import Test"})
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/import?budgetName=Import Test", "")
	suite.Assert().Equal(http.StatusBadRequest, recorder.Code, "Request ID %s, response %s", recorder.Header().Get("x-request-id"), recorder.Body.String())
	suite.Assert().Contains(test.DecodeError(suite.T(), recorder.Body.Bytes()), "This budget name is already in use")

	// Database error. This test must be the last one.
	suite.CloseDB()
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/import?budgetName=Import Test", "")
	suite.Assert().Equal(http.StatusInternalServerError, recorder.Code, "Request ID %s, response %s", recorder.Header().Get("x-request-id"), recorder.Body.String())
	suite.Assert().Contains(test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteStandard) TestImport() {
	// Import one
	body, headers := suite.loadTestFile("Budget.yfull")
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/import?budgetName=Test Budget", body, headers)
	suite.Assert().Equal(http.StatusCreated, recorder.Code, "Request ID %s, response %s", recorder.Header().Get("x-request-id"), recorder.Body.String())
}
