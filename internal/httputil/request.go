package httputil

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// The scheme defaults to https and only falls back to http
// if the x-forwarded-proto header is set to "http".
func RequestHost(c *gin.Context) string {
	scheme := "http"
	if c.Request.Header.Get("x-forwarded-proto") == "https" {
		scheme = "https"
	}

	forwardedPrefix := c.Request.Header.Get("x-forwarded-prefix")
	return scheme + "://" + c.Request.Host + forwardedPrefix
}

// RequestPathV1 returns the URL with the prefix for API v1.
func RequestPathV1(c *gin.Context) string {
	return RequestHost(c) + "/v1"
}

// RequestURL returns the full request URL.
func RequestURL(c *gin.Context) string {
	return RequestHost(c) + c.Request.URL.Path
}

// ParseID parses the ID.
func ParseID(c *gin.Context, param string) (uint64, error) {
	var parsed uint64

	parsed, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		FetchErrorHandler(c, err)
		return 0, err
	}

	return parsed, nil
}

// BindData binds the data from the request to the struct passed in the interface.
func BindData(c *gin.Context, data interface{}) error {
	if err := c.ShouldBindJSON(&data); err != nil {
		if errors.Is(io.EOF, err) {
			e := errors.New("request body must not be emtpy")
			NewError(c, http.StatusBadRequest, e)
			return e
		}

		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		e := errors.New("the body of your request contains invalid or un-parseable data. Please check and try again")
		NewError(c, http.StatusBadRequest, e)
		return e
	}

	return nil
}
