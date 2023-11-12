package httperrors

type HTTPError struct {
	Error string `json:"error" example:"An ID specified in the query string was not a valid UUID"`
}

// Error is used to return an error with the corresponding HTTP status code to a controller.
type Error struct {
	Err    error
	Status int // Used with http.StatusX for the corresponding HTTP status code
}

// Nil checks if the ErrorStatus is the zero value.
func (e Error) Nil() bool {
	return e.Err == nil && e.Status == 0
}

// Error returns the error as a string.
func (e Error) Error() string {
	s := e.Err.Error()
	return s
}
