package controllers_test

import (
	"net/http"

	"github.com/envelope-zero/backend/v3/test"
)

var methodNotAllowedTests = []struct {
	path   string
	method string
}{
	{"/", http.MethodPost},
	{"/", http.MethodDelete},
	{"http://example.com/v1", http.MethodPost},
	{"http://example.com/v1/budgets", "HEAD"},
	{"http://example.com/v1/budgets", "PUT"},
}

func (suite *TestSuiteStandard) TestMethodNotAllowed() {
	for _, tt := range methodNotAllowedTests {
		recorder := test.Request(suite.controller, suite.T(), tt.method, tt.path, "")

		assertHTTPStatus(suite.T(), &recorder, http.StatusMethodNotAllowed)
	}
}
