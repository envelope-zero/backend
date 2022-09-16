package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/router"
	"github.com/stretchr/testify/assert"
)

// TOLERANCE is the number of seconds that a CreatedAt or UpdatedAt time.Time
// is allowed to differ from the time at which it is checked.
//
// As CreatedAt and UpdatedAt are automatically set by gorm, we need a tolerance here.
// This is in nanoseconds, so we multiply by 1000000000 for seconds.
const TOLERANCE time.Duration = 1000000000 * 60

type APIResponse struct {
	Links map[string]string
	Error string
}

// Request is a helper method to simplify making a HTTP request for tests.
func Request(co controllers.Controller, t *testing.T, method, url string, body any, headers ...map[string]string) httptest.ResponseRecorder {
	var byteStr []byte
	var err error

	// If the body is a string, convert it to bytes
	if reflect.TypeOf(body).Kind() == reflect.String {
		byteStr = []byte(body.(string))
	} else {
		byteStr, err = json.Marshal(body)
		if err != nil {
			assert.FailNow(t, "Request body could not be marshalled from object input", err)
		}
	}

	r, err := router.Config()
	if err != nil {
		assert.FailNow(t, "Router could not be initialized")
	}
	router.AttachRoutes(co, r.Group("/"))

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(byteStr))

	for _, headerMap := range headers {
		for header, value := range headerMap {
			req.Header.Set(header, value)
		}
	}

	r.ServeHTTP(recorder, req)

	return *recorder
}

func AssertHTTPStatus(t *testing.T, expected int, r *httptest.ResponseRecorder) {
	assert.Equal(t, expected, r.Code, "HTTP status is wrong. Request ID: '%s' Response body: %s", r.Result().Header.Get("x-request-id"), r.Body.String())
}

// DecodeResponse decodes an HTTP response into a target struct.
func DecodeResponse(t *testing.T, r *httptest.ResponseRecorder, target interface{}) {
	err := json.NewDecoder(r.Body).Decode(target)
	if err != nil {
		assert.FailNow(t, "Parsing error", "Unable to parse response from server %q into %v, '%v', Request ID: %s", r.Body, reflect.TypeOf(target), err, r.Result().Header.Get("x-request-id"))
	}
}

func DecodeError(t *testing.T, s []byte) string {
	var r httperrors.HTTPError
	if err := json.Unmarshal(s, &r); err != nil {
		assert.Fail(t, "Not valid JSON!", "%s", s)
	}

	return r.Error
}
