package httperrors

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/go-sqlite"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

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

// GenericDBError wraps DBError with a more specific error message for not found errors.
func GenericDBError[T models.Model](r T, c *gin.Context, err error) ErrorStatus {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrorStatus{Status: http.StatusNotFound, Err: fmt.Errorf("there is no %s with this ID", r.Self())}
	}

	return DBError(c, err)
}

// DBError returns an error message and status code appropriate to the error that has occurred.
func DBError(c *gin.Context, err error) ErrorStatus {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrorStatus{Status: http.StatusNotFound, Err: errors.New("there is no resource for the ID you specified")}
	}

	// Availability month is set before the month of the transaction
	if strings.Contains(err.Error(), "availability month must not be earlier than the month of the transaction") {
		return ErrorStatus{Status: http.StatusBadRequest, Err: err}
	}

	// Account cannot be on budget because transactions have envelopes
	if strings.Contains(err.Error(), "the account cannot be set to on budget because") {
		return ErrorStatus{Status: http.StatusBadRequest, Err: err}
	}

	// Account name must be unique per Budget
	if strings.Contains(err.Error(), "UNIQUE constraint failed: accounts.name, accounts.budget_id") {
		return ErrorStatus{Status: http.StatusBadRequest, Err: errors.New("the account name must be unique for the budget")}
	}

	// Category names need to be unique per budget
	if strings.Contains(err.Error(), "UNIQUE constraint failed: categories.name, categories.budget_id") {
		return ErrorStatus{Status: http.StatusBadRequest, Err: errors.New("the category name must be unique for the budget")}
	}

	// Unique envelope names per category
	if strings.Contains(err.Error(), "UNIQUE constraint failed: envelopes.name, envelopes.category_id") {
		return ErrorStatus{Status: http.StatusBadRequest, Err: errors.New("the envelope name must be unique for the category")}
	}

	// Only one allocation per envelope per month
	if strings.Contains(err.Error(), "UNIQUE constraint failed: allocations.month, allocations.envelope_id") {
		return ErrorStatus{Status: http.StatusBadRequest, Err: errors.New("you can not create multiple allocations for the same month")}
	}

	// Source and destination accounts need to be different
	if strings.Contains(err.Error(), "CHECK constraint failed: source_destination_different") {
		return ErrorStatus{Status: http.StatusBadRequest, Err: errors.New("source and destination accounts for a transaction must be different")}
	}

	// General message when a field references a non-existing resource
	if strings.Contains(err.Error(), "constraint failed: FOREIGN KEY constraint failed") {
		return ErrorStatus{Status: http.StatusBadRequest, Err: errors.New("there is no resource for the ID you specificed in the reference to another resource")}
	}

	// Database is read only or file has been deleted
	if strings.Contains(err.Error(), "attempt to write a readonly database (1032)") {
		log.Error().Msgf("Database is in read-only mode. This might be due to the file being deleted: %#v", err)
		return ErrorStatus{Status: http.StatusInternalServerError, Err: errors.New("the database is currently in read-only mode, please try again later")}
	}

	// A general error we do not know more about
	log.Error().Msgf("%T: %v", err, err.Error())
	return ErrorStatus{Status: http.StatusInternalServerError, Err: fmt.Errorf("an error occurred on the server during your request, please contact your server administrator. The request id is '%v', send this to your server administrator to help them finding the problem", requestid.Get(c))}
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
		status := DBError(c, err)
		New(c, status.Status, status.Error())
	} else if errors.Is(err, models.ErrAllocationZero) {
		New(c, http.StatusBadRequest, err.Error())

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
