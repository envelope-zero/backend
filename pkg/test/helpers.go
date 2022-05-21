package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/stretchr/testify/assert"
)

// TOLERANCE is the number of seconds that a CreatedAt or UpdatedAt time.Time
// is allowed to differ from the time at which it is checked.
//
// As CreatedAt and UpdatedAt are automatically set by gorm, we need a tolerance here.
const TOLERANCE time.Duration = 1000000000 * 60

type APIResponse struct {
	Links map[string]string
	Error string
}

// Request is a helper method to simplify making a HTTP request for tests.
func Request(t *testing.T, method, url string, body any, headers ...map[string]string) httptest.ResponseRecorder {
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

	os.Setenv("LOG_FORMAT", "human")
	router, err := controllers.Router()
	if err != nil {
		assert.FailNow(t, "Router could not be initialized")
	}

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(byteStr))

	for _, headerMap := range headers {
		for header, value := range headerMap {
			req.Header.Set(header, value)
		}
	}

	router.ServeHTTP(recorder, req)

	return *recorder
}

func AssertHTTPStatus(t *testing.T, expected int, r *httptest.ResponseRecorder) {
	assert.Equal(t, expected, r.Code, "HTTP status is wrong. Response body: %s", r.Body.String())
}

// DecodeResponse decodes an HTTP response into a target struct.
func DecodeResponse(t *testing.T, r *httptest.ResponseRecorder, target interface{}) {
	err := json.NewDecoder(r.Body).Decode(target)
	if err != nil {
		assert.FailNow(t, "Parsing error", "Unable to parse response from server %q into %v, '%v'", r.Body, reflect.TypeOf(target), err)
	}
}

func DecodeError(t *testing.T, s []byte) string {
	var r httputil.HTTPError
	if err := json.Unmarshal(s, &r); err != nil {
		assert.Fail(t, "Not valid JSON!", "%s", s)
	}

	return r.Error
}
