package controllers_test

import (
	"net/http"

	"github.com/envelope-zero/backend/pkg/test"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteEnv) TestOptionsHeaderResources() {
	optionsHeaderTests := []string{
		"http://example.com/v1/budgets",
		"http://example.com/v1/accounts",
		"http://example.com/v1/categories",
		"http://example.com/v1/envelopes",
		"http://example.com/v1/allocations",
		"http://example.com/v1/transactions",
	}

	for _, path := range optionsHeaderTests {
		recorder := test.Request(suite.T(), http.MethodOptions, path, "")

		assert.Equal(suite.T(), http.StatusNoContent, recorder.Code)
		assert.Equal(suite.T(), recorder.Header().Get("allow"), "GET, POST")
	}
}
