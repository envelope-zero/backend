package httputil_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestFetchErrorHandlerErrRecordNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httputil.FetchErrorHandler(c, gorm.ErrRecordNotFound)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFetchErrorHandlerStrconvNumError(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httputil.FetchErrorHandler(c, &strconv.NumError{})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "An ID specified in the query string was not a valid uint64", test.DecodeError(t, w.Body.Bytes()))
}

func TestFetchErrorHandlerTimeParseError(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httputil.FetchErrorHandler(c, &time.ParseError{})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "parsing time")
}

func TestFetchErrorHandlerInternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httputil.FetchErrorHandler(c, errors.New("Some random error"))
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "an error occurred on the server during your request")
}

func TestErrorInvalidUUID(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httputil.ErrorInvalidUUID(c)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "not a valid UUID")
}
