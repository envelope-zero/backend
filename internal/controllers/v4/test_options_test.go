package v4_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v7/test"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestOptionsHeaderResources() {
	optionsHeaderTests := []struct {
		path     string
		response string
	}{
		{"http://example.com/v4", "OPTIONS, GET, DELETE"},
		{"http://example.com/v4/accounts", "OPTIONS, GET, POST"},
		{"http://example.com/v4/budgets", "OPTIONS, GET, POST"},
		{"http://example.com/v4/categories", "OPTIONS, GET, POST"},
		{"http://example.com/v4/envelopes", "OPTIONS, GET, POST"},
		{"http://example.com/v4/export", "OPTIONS, GET"},
		{"http://example.com/v4/goals", "OPTIONS, GET, POST"},
		{"http://example.com/v4/import", "OPTIONS, GET"},
		{"http://example.com/v4/import/ynab-import-preview", "OPTIONS, POST"},
		{"http://example.com/v4/import/ynab4", "OPTIONS, POST"},
		{"http://example.com/v4/match-rules", "OPTIONS, GET, POST"},
		{"http://example.com/v4/months", "OPTIONS, GET, POST, DELETE"},
		{"http://example.com/v4/transactions", "OPTIONS, GET, POST"},
	}

	for _, tt := range optionsHeaderTests {
		suite.T().Run(tt.path, func(t *testing.T) {
			recorder := test.Request(suite.T(), http.MethodOptions, tt.path, "")

			assert.Equal(t, http.StatusNoContent, recorder.Code)
			assert.Equal(t, recorder.Header().Get("allow"), tt.response)
		})
	}
}
