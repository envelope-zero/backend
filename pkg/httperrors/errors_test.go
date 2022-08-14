package httperrors_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestFetchErrorHandlerErrRecordNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.Handler(c, gorm.ErrRecordNotFound)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFetchErrorHandlerTimeParseError(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.Handler(c, &time.ParseError{})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "parsing time")
}

func TestFetchErrorHandlerSQLiteErrorUnknown(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.Handler(c, &sqlite.Error{})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "A database error")
}

func TestFetchErrorHandlerEOF(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.Handler(c, io.EOF)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "The request body must not be empty")
}

func TestFetchErrorHandlerInternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.Handler(c, errors.New("Some random error"))
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "An error occurred on the server during your request")
}

func TestErrorInvalidUUID(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.InvalidUUID(c)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "not a valid UUID")
}

func TestErrorUnparseableData(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.InvalidQueryString(c)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "unparseable data")
}

func TestErrorInvalidMonth(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.InvalidMonth(c)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "did you use YYYY-MM format?")
}

func TestDatabaseErrorMessages(t *testing.T) {
	tests := []struct {
		code int
		err  string
		msg  string
	}{
		{http.StatusBadRequest, "CHECK constraint failed: source_destination_different", "Source and destination accounts for a transaction must be different"},
		{http.StatusBadRequest, "UNIQUE constraint failed: categories.name, categories.budget_id", "The category name must be unique for the budget"},
		{http.StatusBadRequest, "UNIQUE constraint failed: envelopes.name, envelopes.category_id", "The envelope name must be unique for the category"},
		{http.StatusBadRequest, "UNIQUE constraint failed: allocations.month, allocations.envelope_id", "You can not create multiple allocations for the same month"},
		{http.StatusBadRequest, "constraint failed: FOREIGN KEY constraint failed", "A resource ID you specfied did not identify an existing resource."},
		{http.StatusInternalServerError, "This is a very weird error", "A database error occurred during your request"},
	}

	for _, tt := range tests {
		err := errors.New(tt.err)
		code, msg := httperrors.DBErrorMessage(err)
		assert.Equal(t, tt.code, code)
		assert.Equal(t, tt.msg, msg)
	}
}
