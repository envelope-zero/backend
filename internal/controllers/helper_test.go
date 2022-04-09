package controllers_test

import (
	"testing"

	"github.com/envelope-zero/backend/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestRequestURLHTTPS(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1", "", map[string]string{"x-forwarded-proto": "https"})

	var apiResponse test.APIResponse
	test.DecodeResponse(t, &recorder, &apiResponse)

	assert.Equal(t, "https:///v1/budgets", apiResponse.Links["budgets"])
}

func TestRequestForwardedPrefix(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1", "", map[string]string{"x-forwarded-prefix": "/api"})

	var apiResponse test.APIResponse
	test.DecodeResponse(t, &recorder, &apiResponse)

	assert.Equal(t, "http:///api/v1/budgets", apiResponse.Links["budgets"])
}
