package httperrors

type HTTPError struct {
	Error string `json:"error" example:"An ID specified in the query string was not a valid UUID"`
}

// ErrorStatus is used to return an error with the corresponding HTTP status code to a controller.
type ErrorStatus struct {
	Err    error
	Status int // Used with http.StatusX for the corresponding HTTP status code
}

// Nil checks if the ErrorStatus is the zero value.
func (e ErrorStatus) Nil() bool {
	return e.Err == nil && e.Status == 0
}

// Body returns the the content of the HTTP response body for the error.
func (e ErrorStatus) Body() map[string]string {
	return map[string]string{
		"error": e.Error(),
	}
}

// Error returns the error as string.
func (e ErrorStatus) Error() string {
	return e.Err.Error()
}
