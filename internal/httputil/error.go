package httputil

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// FetchErrorHandler handles errors for fetching data from the database.
func FetchErrorHandler(c *gin.Context, err error) {
	// No record found => 404
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusNotFound)

		// Number parsing error => 400
	} else if reflect.TypeOf(err) == reflect.TypeOf(&strconv.NumError{}) {
		NewError(c, http.StatusBadRequest, errors.New("An ID specified in the query string was not a valid uint64"))

		// Time Parsing error => 400
	} else if reflect.TypeOf(err) == reflect.TypeOf(&time.ParseError{}) {
		NewError(c, http.StatusBadRequest, err)

		// All other errors
	} else {
		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		NewError(c, http.StatusInternalServerError, fmt.Errorf("an error occured on the server during your request, please contact your server administrator. The request id is '%v', send this to your server administrator to help them finding the problem", requestid.Get(c)))
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
