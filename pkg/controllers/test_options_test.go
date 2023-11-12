package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v3/test"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestOptionsHeaderResources() {
	optionsHeaderTests := []struct {
		path     string
		response string
	}{
		{"http://example.com/healthz", "OPTIONS, GET"},
		{"http://example.com/v1/budgets", "OPTIONS, GET, POST"},
		{"http://example.com/v1/accounts", "OPTIONS, GET, POST"},
		{"http://example.com/v1/categories", "OPTIONS, GET, POST"},
		{"http://example.com/v1/envelopes", "OPTIONS, GET, POST"},
		{"http://example.com/v1/allocations", "OPTIONS, GET, POST"},
		{"http://example.com/v1/transactions", "OPTIONS, GET, POST"},
		{"http://example.com/v1/month-configs", "OPTIONS, GET"},
		{"http://example.com/v1/import", "OPTIONS, POST"},
		{"http://example.com/v1/import/ynab4", "OPTIONS, POST"},
		{"http://example.com/v1/import/ynab-import-preview", "OPTIONS, POST"},
		{"http://example.com/v2/transactions", "OPTIONS, POST"},
		{"http://example.com/v2/match-rules", "OPTIONS, GET, POST"},
		{"http://example.com/v2/accounts", "OPTIONS, GET"},
		{"http://example.com/v3/transactions", "OPTIONS, GET, POST"},
	}

	for _, tt := range optionsHeaderTests {
		suite.T().Run(tt.path, func(t *testing.T) {
			recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, tt.path, "")

			assert.Equal(t, http.StatusNoContent, recorder.Code)
			assert.Equal(t, recorder.Header().Get("allow"), tt.response)
		})
	}
}
