package models

import (
	"errors"
)

var (
	ErrGeneral          = errors.New("an error occurred on the server during your request")
	ErrResourceNotFound = errors.New("there is no")
)
