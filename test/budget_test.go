package test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoBudgets(t *testing.T) {
	recorder := Request("GET", "/v1/budgets", nil)

	require.Equal(t, 200, recorder.Code)
	require.JSONEq(t, `{ "data": [] }`, recorder.Body.String())
}

func TestNoBudgetNotFound(t *testing.T) {
	recorder := Request("GET", "/v1/budgets/2", nil)

	require.Equal(t, 404, recorder.Code)
}
