package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/envelope-zero/backend/v2/pkg/controllers"
	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/envelope-zero/backend/v2/pkg/router"
	"github.com/stretchr/testify/assert"
)

type APIResponse struct {
	Links map[string]string
	Error string
}

// Request is a helper method to simplify making a HTTP request for tests.
func Request(co controllers.Controller, t *testing.T, method, url string, body any, headers ...map[string]string) httptest.ResponseRecorder {
	var byteBuffer *bytes.Buffer
	var err error

	// If the body is a string, convert it to bytes
	if reflect.TypeOf(body).Kind() == reflect.String {
		byteBuffer = bytes.NewBufferString(body.(string))
	} else if reflect.TypeOf(body).Kind() == reflect.Struct || reflect.TypeOf(body).Kind() == reflect.Map {
		byteStr, err := json.Marshal(body)
		if err != nil {
			assert.Fail(t, "Request body could not be marshalled from struct input", err)
		}
		byteBuffer = bytes.NewBuffer(byteStr)
	} else {
		// Assume we got sent a *bytes.Buffer for e.g. a file
		byteBuffer = body.(*bytes.Buffer)
	}

	r, err := router.Config()
	if err != nil {
		assert.FailNow(t, "Router could not be initialized")
	}
	router.AttachRoutes(co, r.Group("/"))

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, byteBuffer)

	for _, headerMap := range headers {
		for header, value := range headerMap {
			req.Header.Set(header, value)
		}
	}

	r.ServeHTTP(recorder, req)

	return *recorder
}

func DecodeError(t *testing.T, s []byte) string {
	var r httperrors.HTTPError
	if err := json.Unmarshal(s, &r); err != nil {
		assert.Fail(t, "Not valid JSON!", "%s", s)
	}

	return r.Error
}
