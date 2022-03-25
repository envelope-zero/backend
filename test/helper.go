package test

import (
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/envelope-zero/backend/internal/routing"
)

// Request is a helper method to simplify making a HTTP request for tests.
func Request(method string, url string, body io.Reader) httptest.ResponseRecorder {
	router := routing.Router()

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, body)
	router.ServeHTTP(recorder, req)

	return *recorder
}
