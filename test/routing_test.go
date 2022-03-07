package test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoutingRootOverview(t *testing.T) {
	recorder := Request("GET", "/", nil)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{ "v1": "/v1" }`, recorder.Body.String())
}

func TestRoutingOptionsV1(t *testing.T) {
	recorder := Request("OPTIONS", "/", nil)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, recorder.Header().Get("Allow"), "GET")
}
