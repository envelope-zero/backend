package controllers_test

import (
	"encoding/json"
	"testing"

	"github.com/envelope-zero/backend/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestRequestURLHTTPS(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1", "", map[string]string{"x-forwarded-proto": "https"})

	var apiResponse test.APIResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiResponse)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, "https:///v1/budgets", apiResponse.Links["budgets"])
}

func TestRequestForwardedPrefix(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1", "", map[string]string{"x-forwarded-prefix": "/api"})

	var apiResponse test.APIResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiResponse)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, "http:///api/v1/budgets", apiResponse.Links["budgets"])
}
