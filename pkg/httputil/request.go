package httputil

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// The scheme defaults to https and only falls back to http
// if the x-forwarded-proto header is set to "http".
func RequestHost(c *gin.Context) string {
	// Default to the URL scheme. This might be empty du to http.Request being weird
	scheme := c.Request.URL.Scheme

	if c.Request.Header.Get("x-forwarded-proto") != "" {
		scheme = c.Request.Header.Get("x-forwarded-proto")
	}

	if scheme == "" {
		scheme = "http"
	}

	// We can reasonably expect a reverse proxy to set x-forwarded-host
	// as it is a de-facto standard.
	//
	// If it is set, we use it to construct the links and use the
	// x-forwarded-prefix header as prefix. If that is unset,
	// fall back to "/api"
	//
	// If no proxy is detected, don’t do anything.
	host := c.Request.Host
	var forwardedPrefix string

	xForwardedHost := c.Request.Header.Get("x-forwarded-host")
	if xForwardedHost != "" {
		host = xForwardedHost

		forwardedPrefix = c.Request.Header.Get("x-forwarded-prefix")

		if forwardedPrefix == "" {
			forwardedPrefix = "/api"
		}
	}

	return scheme + "://" + host + forwardedPrefix
}

// RequestPathV1 returns the URL with the prefix for API v1.
func RequestPathV1(c *gin.Context) string {
	return RequestHost(c) + "/v1"
}

// RequestURL returns the full request URL.
func RequestURL(c *gin.Context) string {
	return RequestHost(c) + c.Request.URL.Path
}

// BindData binds the data from the request to the struct passed in the interface.
func BindData(c *gin.Context, data interface{}) error {
	if err := c.ShouldBindJSON(&data); err != nil {
		if errors.Is(io.EOF, err) {
			e := errors.New("request body must not be empty")
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
