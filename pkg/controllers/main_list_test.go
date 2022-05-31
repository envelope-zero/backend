package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/pkg/test"
)

var methodNotAllowedTests = []struct {
	path   string
	method string
}{
	{"/", "POST"},
	{"/", "DELETE"},
	{"http://example.com/v1", "POST"},
	{"http://example.com/v1", "DELETE"},
	{"http://example.com/v1/budgets", "HEAD"},
	{"http://example.com/v1/budgets", "PUT"},
}

func TestMethodNotAllowed(t *testing.T) {
	for _, tt := range methodNotAllowedTests {
		recorder := test.Request(t, tt.method, tt.path, "")

		test.AssertHTTPStatus(t, http.StatusMethodNotAllowed, &recorder)
	}
}
