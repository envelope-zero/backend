package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/pkg/test"
	"github.com/stretchr/testify/assert"
)

var optionsHeaderTests = []struct {
	path     string
	expected string
}{
	{"/", "GET"},
	{"/version", "GET"},
	{"/v1", "GET"},
	{"/v1/budgets", "GET, POST"},
	{"/v1/budgets/1", "GET, PATCH, DELETE"},
	{"/v1/accounts", "GET, POST"},
	{"/v1/accounts/1", "GET, PATCH, DELETE"},
	{"/v1/categories", "GET, POST"},
	{"/v1/categories/1", "GET, PATCH, DELETE"},
	{"/v1/envelopes", "GET, POST"},
	{"/v1/envelopes/1", "GET, PATCH, DELETE"},
	{"/v1/allocations", "GET, POST"},
	{"/v1/allocations/1", "GET, PATCH, DELETE"},
	{"/v1/transactions", "GET, POST"},
	{"/v1/transactions/1", "GET, PATCH, DELETE"},
}

func TestOptionsHeader(t *testing.T) {
	for _, tt := range optionsHeaderTests {
		recorder := test.Request(t, "OPTIONS", tt.path, "")

		assert.Equal(t, http.StatusNoContent, recorder.Code, "Status for %v is wrong, expected %d, got %d", tt.path, http.StatusNoContent, recorder.Code)
		assert.Equal(t, recorder.Header().Get("allow"), tt.expected)
	}
}
