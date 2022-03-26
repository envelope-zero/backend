package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/internal/test"
	"github.com/stretchr/testify/assert"
)

var optionsHeaderTests = []struct {
	path     string
	expected string
}{
	{"/", "GET"},
	{"/v1", "GET"},
	{"/v1/budgets", "GET, POST"},
	{"/v1/budgets/1", "GET, PATCH, DELETE"},
	{"/v1/budgets/1/accounts", "GET, POST"},
	{"/v1/budgets/1/accounts/1", "GET, PATCH, DELETE"},
	{"/v1/budgets/1/accounts/1/transactions", "GET"},
	{"/v1/budgets/1/categories", "GET, POST"},
	{"/v1/budgets/1/categories/1", "GET, PATCH, DELETE"},
	{"/v1/budgets/1/categories/1/envelopes", "GET, POST"},
	{"/v1/budgets/1/categories/1/envelopes/1", "GET, PATCH, DELETE"},
	{"/v1/budgets/1/categories/1/envelopes/1/allocations", "GET, POST"},
	{"/v1/budgets/1/categories/1/envelopes/1/allocations/1", "GET, PATCH, DELETE"},
	{"/v1/budgets/1/transactions", "GET, POST"},
	{"/v1/budgets/1/transactions/1", "GET, PATCH, DELETE"},
}

func TestOptionsHeader(t *testing.T) {
	for _, tt := range optionsHeaderTests {
		recorder := test.Request(t, "OPTIONS", tt.path, "")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, recorder.Header().Get("allow"), tt.expected)
	}
}
