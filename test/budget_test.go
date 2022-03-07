package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoBudgets(t *testing.T) {
	recorder := Request("GET", "/v1/budgets", nil)

	assert.Equal(t, 200, recorder.Code)
	assert.JSONEq(t, `{ "data": [] }`, recorder.Body.String())
}

func TestNoBudgetNotFound(t *testing.T) {
	recorder := Request("GET", "/v1/budgets/2", nil)

	assert.Equal(t, 404, recorder.Code)
}
