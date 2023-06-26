package httperrors_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/stretchr/testify/assert"
)

func TestErrorStatusNil(t *testing.T) {
	err := httperrors.ErrorStatus{}
	assert.True(t, err.Nil())
}

func TestErrorStatusNotNil(t *testing.T) {
	err := httperrors.ErrorStatus{
		Status: http.StatusOK,
	}
	assert.False(t, err.Nil())
}

func TestErrorStatusBody(t *testing.T) {
	err := httperrors.ErrorStatus{
		Status: http.StatusBadRequest,
		Err:    errors.New("Testing the response body"),
	}
	assert.Equal(t, map[string]string{
		"error": "Testing the response body",
	}, err.Body())
}
