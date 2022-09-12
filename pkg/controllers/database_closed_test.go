package controllers_test

import (
	"net/http"

	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/test"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteClosedDB) TestBudgets() {
	recorder := test.Request(suite.T(), http.MethodPost, "http://example.com/v1/budgets", models.BudgetCreate{})
	test.AssertHTTPStatus(suite.T(), http.StatusInternalServerError, &recorder)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")

	recorder = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets", "")
	test.AssertHTTPStatus(suite.T(), http.StatusInternalServerError, &recorder)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteClosedDB) TestAccounts() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/accounts", "")
	test.AssertHTTPStatus(suite.T(), http.StatusInternalServerError, &recorder)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteClosedDB) TestAllocations() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/allocations", "")
	test.AssertHTTPStatus(suite.T(), http.StatusInternalServerError, &recorder)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteClosedDB) TestCategories() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/categories", "")
	test.AssertHTTPStatus(suite.T(), http.StatusInternalServerError, &recorder)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteClosedDB) TestEnvelopes() {
	recorder := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/envelopes", "")
	test.AssertHTTPStatus(suite.T(), http.StatusInternalServerError, &recorder)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}
