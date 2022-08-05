package httputil

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/go-sqlite"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// ErrorHandler handles errors for fetching data from the database.
func ErrorHandler(c *gin.Context, err error) {
	// No record found => 404
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusNotFound)

		// Database error
	} else if reflect.TypeOf(err) == reflect.TypeOf(&sqlite.Error{}) {
		if strings.Contains(err.Error(), "constraint failed: FOREIGN KEY constraint failed") {
			NewError(c, http.StatusBadRequest, errors.New("A resource ID you specfied did not identify an existing resource"))
		} else if strings.Contains(err.Error(), "CHECK constraint failed: source_destination_different") {
			NewError(c, http.StatusBadRequest, errors.New("Source and destination accounts for a transaction must be different"))
		} else if strings.Contains(err.Error(), "CHECK constraint failed: month_valid") {
			NewError(c, http.StatusBadRequest, errors.New("The month must be between 1 and 12"))
		} else if strings.Contains(err.Error(), "UNIQUE constraint failed: allocations.month") {
			NewError(c, http.StatusBadRequest, errors.New("You can not create multiple allocations for the same month"))
		} else {
			log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
			NewError(c, http.StatusInternalServerError, fmt.Errorf("A database error occured during your reuqest, please contact your server administrator. The request id is '%v', send this to your server administrator to help them finding the problem", requestid.Get(c)))
		}

		// End of file reached when reading
	} else if errors.Is(io.EOF, err) {
		NewError(c, http.StatusBadRequest, errors.New("request body must not be empty"))

		// Number parsing error => 400
	} else if reflect.TypeOf(err) == reflect.TypeOf(&strconv.NumError{}) {
		NewError(c, http.StatusBadRequest, errors.New("An ID specified in the query string was not a valid uint64"))

		// Time Parsing error => 400
	} else if reflect.TypeOf(err) == reflect.TypeOf(&time.ParseError{}) {
		NewError(c, http.StatusBadRequest, err)

		// All other errors
	} else {
		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		NewError(c, http.StatusInternalServerError, fmt.Errorf("an error occurred on the server during your request, please contact your server administrator. The request id is '%v', send this to your server administrator to help them finding the problem", requestid.Get(c)))
	}
}

// NewError creates an HTTPError instance and returns it.
func NewError(c *gin.Context, status int, err error) {
	e := HTTPError{
		Error: err.Error(),
	}
	c.JSON(status, e)
}

// HTTPError is used for error responses that contain a body.
type HTTPError struct {
	Error string `json:"error" example:"An ID specified in the query string was not a valid uint64"`
}

func ErrorInvalidUUID(c *gin.Context) {
	NewError(c, http.StatusBadRequest, errors.New("The specified resource ID is not a valid UUID"))
}

func ErrorInvalidQueryString(c *gin.Context) {
	NewError(c, http.StatusBadRequest, errors.New("The query string contains unparseable data. Please check the values"))
}
