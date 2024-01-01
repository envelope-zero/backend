package httperrors_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/stretchr/testify/assert"
)

func TestErrorStatusNil(t *testing.T) {
	err := httperrors.Error{}
	assert.True(t, err.Nil())
}

func TestErrorStatusNotNil(t *testing.T) {
	err := httperrors.Error{
		Status: http.StatusOK,
	}
	assert.False(t, err.Nil())
}
