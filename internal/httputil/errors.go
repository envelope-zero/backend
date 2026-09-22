package httputil

import "errors"

var (
	ErrInvalidBody      = errors.New("the body of your request contains invalid or un-parseable data. Please check and try again")
	ErrRequestBodyEmpty = errors.New("the request body must not be empty")
	ErrInvalidUUID      = errors.New("the specified resource ID is not a valid UUID")
)
