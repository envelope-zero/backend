package controllers_test

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"time"

	"github.com/stretchr/testify/assert"
)

// TOLERANCE is the number of seconds that a CreatedAt or UpdatedAt time.Time
// is allowed to differ from the time at which it is checked.
//
// As CreatedAt and UpdatedAt are automatically set by gorm, we need a tolerance here.
// This is in nanoseconds, so we multiply by 1000000000 for seconds.
const tolerance time.Duration = 1000000000 * 60

func (suite *TestSuiteStandard) assertHTTPStatus(r *httptest.ResponseRecorder, expectedStatus ...int) {
	assert.Contains(suite.T(), expectedStatus, r.Code, "HTTP status is wrong. Request ID: '%s' Response body: %s", r.Result().Header.Get("x-request-id"), r.Body.String())
}

// decodeResponse decodes an HTTP response into a target struct.
func (suite *TestSuiteStandard) decodeResponse(r *httptest.ResponseRecorder, target interface{}) {
	err := json.NewDecoder(r.Body).Decode(target)
	if err != nil {
		assert.FailNow(suite.T(), "Parsing error", "Unable to parse response from server %q into %v, '%v', Request ID: %s", r.Body, reflect.TypeOf(target), err, r.Result().Header.Get("x-request-id"))
	}
}
