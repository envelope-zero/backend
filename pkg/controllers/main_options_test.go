package controllers_test

import (
	"net/http"

	"github.com/envelope-zero/backend/pkg/test"
	"github.com/stretchr/testify/assert"
)

var optionsHeaderTests = []struct {
	path     string
	expected string
}{
	{"/", http.MethodGet},
	{"/version", http.MethodGet},
	{"http://example.com/v1", http.MethodGet},
	{"http://example.com/v1/budgets", "GET, POST"},
	{"http://example.com/v1/budgets/1", "GET, PATCH, DELETE"},
	{"http://example.com/v1/accounts", "GET, POST"},
	{"http://example.com/v1/accounts/1", "GET, PATCH, DELETE"},
	{"http://example.com/v1/categories", "GET, POST"},
	{"http://example.com/v1/categories/1", "GET, PATCH, DELETE"},
	{"http://example.com/v1/envelopes", "GET, POST"},
	{"http://example.com/v1/envelopes/1", "GET, PATCH, DELETE"},
	{"http://example.com/v1/allocations", "GET, POST"},
	{"http://example.com/v1/allocations/1", "GET, PATCH, DELETE"},
	{"http://example.com/v1/transactions", "GET, POST"},
	{"http://example.com/v1/transactions/1", "GET, PATCH, DELETE"},
}

func (suite *TestSuiteEnv) TestOptionsHeader() {
	for _, tt := range optionsHeaderTests {
		recorder := test.Request(suite.T(), http.MethodOptions, tt.path, "")

		assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Status for %v is wrong, expected %d, got %d", tt.path, http.StatusNoContent, recorder.Code)
		assert.Equal(suite.T(), recorder.Header().Get("allow"), tt.expected)
	}
}
