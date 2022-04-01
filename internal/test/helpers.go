package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/controllers"
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
func Request(t *testing.T, method, url, body string, headers ...map[string]string) httptest.ResponseRecorder {
	byteStr := []byte(body)

	router, err := controllers.Router()
	if err != nil {
		t.Fatal(err)
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
	assert.Equal(t, expected, r.Code, "Status is '%v', body is '%v'", r.Code, r.Body.String())
}
