package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/envelope-zero/backend/v5/pkg/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Request is a helper method to simplify making a HTTP request for tests.
func Request(t *testing.T, method, reqURL string, body any, headers ...map[string]string) httptest.ResponseRecorder {
	var byteBuffer *bytes.Buffer
	var err error

	// If the body is a string, convert it to bytes
	if reflect.TypeOf(body).Kind() == reflect.String {
		byteBuffer = bytes.NewBufferString(body.(string))
	} else if reflect.TypeOf(body).Kind() == reflect.Struct || reflect.TypeOf(body).Kind() == reflect.Map || reflect.TypeOf(body).Kind() == reflect.Slice {
		byteStr, err := json.Marshal(body)
		if err != nil {
			assert.Fail(t, "Request body could not be marshalled from struct input", err)
		}
		byteBuffer = bytes.NewBuffer(byteStr)
	} else {
		// Assume we got sent a *bytes.Buffer for e.g. a file
		byteBuffer = body.(*bytes.Buffer)
	}

	apiURL, ok := os.LookupEnv("API_URL")
	if !ok {
		assert.FailNow(t, "environment variable API_URL must be set")
	}

	baseURL, err := url.Parse(apiURL)
	if err != nil {
		assert.FailNow(t, "environment variable API_URL must be a valid URL")
	}

	r, teardown, err := router.Config(baseURL)
	defer teardown()

	if err != nil {
		assert.FailNow(t, "Router could not be initialized")
	}
	router.AttachRoutes(r.Group("/"))

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(method, reqURL, byteBuffer)

	for _, headerMap := range headers {
		for header, value := range headerMap {
			req.Header.Set(header, value)
		}
	}

	r.ServeHTTP(recorder, req)

	return *recorder
}

// DecodeResponse decodes an HTTP response into a target struct.
func DecodeResponse(t *testing.T, r *httptest.ResponseRecorder, target any) {
	err := json.Unmarshal(r.Body.Bytes(), &target)
	if err != nil {
		assert.FailNow(t, "Parsing error", "Unable to parse response from server %q into %v, '%v', Request ID: %s", r.Body, reflect.TypeOf(target), err, r.Result().Header.Get("x-request-id"))
	}
}

// AssertHTTPStatus verifies that the HTTP response status is correct
func AssertHTTPStatus(t *testing.T, r *httptest.ResponseRecorder, expectedStatus ...int) {
	require.Contains(t, expectedStatus, r.Code, "HTTP status is wrong. Request ID: '%s' Response body: %s", r.Result().Header.Get("x-request-id"), r.Body.String())
}
