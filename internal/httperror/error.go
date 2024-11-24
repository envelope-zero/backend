package httperror

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v5/pkg/models"
)

type Error struct {
	Message string `json:"error" example:"You must specify a transaction ID"`
}

func New(e error) Error {
	return Error{
		Message: e.Error(),
	}
}

func NewFromString(e string) Error {
	return Error{
		Message: e,
	}
}

func Status(err error) int {
	if errors.Is(err, models.ErrGeneral) {
		return http.StatusInternalServerError
	}

	if errors.Is(err, models.ErrResourceNotFound) {
		return http.StatusNotFound
	}

	return http.StatusBadRequest
}
