package test_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/internal/test"
)

func TestRequest(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1", "", map[string]string{"x-helper-id": "17481"})
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)
}

func TestDecodeResponse(t *testing.T) {
	var budgets test.APIResponse

	r := test.Request(t, "GET", "/v1/budgets", "")
	test.DecodeResponse(t, &r, &budgets)
}
