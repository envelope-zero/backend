package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/test"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestOptionsHeaderResources() {
	optionsHeaderTests := []struct {
		path     string
		response string
	}{
		{"http://example.com/v1/budgets", "OPTIONS, GET, POST"},
		{"http://example.com/v1/accounts", "OPTIONS, GET, POST"},
		{"http://example.com/v1/categories", "OPTIONS, GET, POST"},
		{"http://example.com/v1/envelopes", "OPTIONS, GET, POST"},
		{"http://example.com/v1/allocations", "OPTIONS, GET, POST"},
		{"http://example.com/v1/transactions", "OPTIONS, GET, POST"},
		{"http://example.com/v1/month-configs", "OPTIONS, GET"},
	}

	for _, tt := range optionsHeaderTests {
		suite.T().Run(tt.path, func(t *testing.T) {
			recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, tt.path, "")

			assert.Equal(suite.T(), http.StatusNoContent, recorder.Code)
			assert.Equal(suite.T(), recorder.Header().Get("allow"), tt.response)
		})
	}
}
