package test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoutingRootOverview(t *testing.T) {
	recorder := Request("GET", "/", nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{ "v1": "/v1" }`, recorder.Body.String())
}

func TestRoutingOptionsV1(t *testing.T) {
	recorder := Request("OPTIONS", "/", nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, recorder.Header().Get("Allow"), "GET")
}
