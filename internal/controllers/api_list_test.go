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
	{"/", `{ "v1": "/v1" }`},
	{"/v1", `{ "budgets": "/budgets" }`},
}

func TestGetOverview(t *testing.T) {
	for _, tt := range getOverviewTests {
		recorder := test.Request(t, "GET", tt.path, "")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, tt.expected, recorder.Body.String())
	}
}
