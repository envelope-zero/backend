package v4_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v7/test"
)

// TestMethodNotAllowed tests some endpoints with disallowed HTTP methods
// to verify that the HTTP 405 - Method Not Allowed status is returned
// correctly
func (suite *TestSuiteStandard) TestMethodNotAllowed() {
	tests := []struct {
		path   string
		method string
	}{
		{"/", http.MethodPost},
		{"/", http.MethodDelete},
		{"http://example.com/v4", http.MethodPost},
		{"http://example.com/v4/budgets", http.MethodHead},
		{"http://example.com/v4/budgets", http.MethodPut},
	}

	for _, tt := range tests {
		suite.T().Run(fmt.Sprintf("%s - %s", tt.path, tt.method), func(t *testing.T) {
			recorder := test.Request(t, tt.method, tt.path, "")

			test.AssertHTTPStatus(t, &recorder, http.StatusMethodNotAllowed)
		})
	}
}
