package httperrors

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/go-sqlite"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type HTTPError struct {
	Error string `json:"error" example:"An ID specified in the query string was not a valid uint64"`
}

// Generate a struct containing the HTTP error on the fly.
func New(c *gin.Context, status int, msgAndArgs ...any) {
	// Format msgAndArgs in a final string.
	// This is taken almost exactly from https://github.com/stretchr/testify/blob/181cea6eab8b2de7071383eca4be32a424db38dd/assert/assertions.go#L181
	msg := ""
	if len(msgAndArgs) == 1 {
		if msgAsStr, ok := msgAndArgs[0].(string); ok {
			msg = msgAsStr
		}
		msg = fmt.Sprintf("%+v", msg)
	}

	if len(msgAndArgs) > 1 {
		msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}

	c.JSON(status, HTTPError{
		Error: msg,
	})
}

func InvalidUUID(c *gin.Context) {
	New(c, http.StatusBadRequest, "The specified resource ID is not a valid UUID")
}

func InvalidQueryString(c *gin.Context) {
	New(c, http.StatusBadRequest, "The query string contains unparseable data. Please check the values")
}

func InvalidMonth(c *gin.Context) {
	New(c, http.StatusBadRequest, "Could not parse the specified month, did you use YYYY-MM format?")
}

// DBErrorMessage returns an error message and status code appropriate to the error that has occurred.
func DBErrorMessage(err error) (int, string) {
	// Source and destination accounts need to be different
	if strings.Contains(err.Error(), "CHECK constraint failed: source_destination_different") {
		return http.StatusBadRequest, "Source and destination accounts for a transaction must be different"

		// Category names need to be unique per budget
	} else if strings.Contains(err.Error(), "UNIQUE constraint failed: categories.name, categories.budget_id") {
		return http.StatusBadRequest, "The category name must be unique for the budget"

		// Envelope names need to be unique per category
	} else if strings.Contains(err.Error(), "UNIQUE constraint failed: envelopes.name, envelopes.category_id") {
		return http.StatusBadRequest, "The envelope name must be unique for the category"

		// Only one allocation per envelope per month
	} else if strings.Contains(err.Error(), "UNIQUE constraint failed: allocations.month, allocations.envelope_id") {
		return http.StatusBadRequest, "You can not create multiple allocations for the same month"

		// General message when a field references a non-existing resource
	} else if strings.Contains(err.Error(), "constraint failed: FOREIGN KEY constraint failed") {
		return http.StatusBadRequest, "There is no resource for the ID you specificed in the reference to another resource."

		// Database is read only or file has been deleted
	} else if strings.Contains(err.Error(), "attempt to write a readonly database (1032)") {
		log.Error().Msgf("Database is in read-only mode. This might be due to the file being deleted: %#v", err)
		return http.StatusInternalServerError, "The database is currently in read-only mode, please try again later."

		// A general error we do not know more about
	} else {
		log.Error().Msgf("%T: %v", err, err.Error())
		return http.StatusInternalServerError, "A database error occurred during your request"
	}
}

// Handler handles errors for fetching data from the database.
func Handler(c *gin.Context, err error, notFoundMsg ...string) {
	// No record found => 404
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Allow the specification of more exact messages when no resource is found
		msg := "There is no resource for the ID you specified"
		if len(notFoundMsg) > 0 {
			msg = notFoundMsg[0]
		}

		New(c, http.StatusNotFound, msg)

		// Database error
	} else if reflect.TypeOf(err) == reflect.TypeOf(&sqlite.Error{}) {
		code, msg := DBErrorMessage(err)
		New(c, code, msg)

		// Database connection has not been opened or has been closed already
	} else if strings.Contains(err.Error(), "sql: database is closed") {
		log.Error().Msgf("Database connection is closed: %#v", err)
		New(c, http.StatusInternalServerError, "There is a problem with the database connection, please try again later.")

		// End of file reached when reading
	} else if errors.Is(io.EOF, err) {
		New(c, http.StatusBadRequest, "The request body must not be empty")

		// Time could not be parsed. Return the error string as tells
		// the problem very well
	} else if reflect.TypeOf(err) == reflect.TypeOf(&time.ParseError{}) {
		New(c, http.StatusBadRequest, err.Error())

		// All other errors
	} else {
		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		New(c, http.StatusInternalServerError, fmt.Sprintf("An error occurred on the server during your request, please contact your server administrator. The request id is '%v', send this to your server administrator to help them finding the problem", requestid.Get(c)))
	}
}
