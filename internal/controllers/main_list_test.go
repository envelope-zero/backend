package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/internal/test"
	"github.com/stretchr/testify/assert"
)

var getOverviewTests = []struct {
	path     string
	expected string
}{
	{"/", `{ "links": { "v1": "http:///v1", "version": "http:///version", "docs": "http:///docs/index.html" }}`},
	{"/v1", `{ "links": { "budgets": "http:///v1/budgets", "accounts": "http:///v1/accounts", "transactions": "http:///v1/transactions", "categories": "http:///v1/categories", "envelopes": "http:///v1/envelopes", "allocations": "http:///v1/allocations" }}`},
	{"/version", `{"data": { "version": "0.0.0" }}`},
}

func TestGetOverview(t *testing.T) {
	for _, tt := range getOverviewTests {
		recorder := test.Request(t, "GET", tt.path, "")

		test.AssertHTTPStatus(t, http.StatusOK, &recorder)
		assert.JSONEq(t, tt.expected, recorder.Body.String())
	}
}

var methodNotAllowedTests = []struct {
	path   string
	method string
}{
	{"/", "POST"},
	{"/", "DELETE"},
	{"/v1", "POST"},
	{"/v1", "DELETE"},
	{"/v1/budgets", "HEAD"},
	{"/v1/budgets", "PUT"},
}

func TestMethodNotAllowed(t *testing.T) {
	for _, tt := range methodNotAllowedTests {
		recorder := test.Request(t, tt.method, tt.path, "")

		test.AssertHTTPStatus(t, http.StatusMethodNotAllowed, &recorder)
	}
}
