package httperrors_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/go-sqlite"
	"github.com/shopspring/decimal"
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

func TestDBErrorErrRecordNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	status := httperrors.DBError(c, gorm.ErrRecordNotFound)
	assert.Equal(t, http.StatusNotFound, status.Status)
	assert.Equal(t, "there is no resource for the ID you specified", status.Error())
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
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "an error occurred on the server", w.Body.String())
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

func TestFetchErrorHandlerDatabaseClosed(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.Handler(c, errors.New("sql: database is closed"))
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "problem with the database connection")
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

// TestGenericDBError verifies that the GenericDBError reuturns a HTTP 404 if a resource does not exist.
func TestGenericDBError(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		err := httperrors.GenericDBError[models.Account](models.Account{}, c, gorm.ErrRecordNotFound)
		assert.False(t, err.Nil())
		assert.Equal(t, http.StatusNotFound, err.Status)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
}

func TestDatabaseErrorMessages(t *testing.T) {
	tests := []struct {
		code int
		err  string
		msg  string
	}{
		{http.StatusBadRequest, "availability month must not be earlier than the month of the transaction, transaction date: 2023-10-22, available month 2023-09", "availability month must not be earlier than the month of the transaction, transaction date: 2023-10-22, available month 2023-09"},
		{http.StatusBadRequest, "the account cannot be set to on budget because the following transactions have an envelope set", "the account cannot be set to on budget because the following transactions have an envelope set"},
		{http.StatusBadRequest, "CHECK constraint failed: source_destination_different", "source and destination accounts for a transaction must be different"},
		{http.StatusBadRequest, "UNIQUE constraint failed: accounts.name, accounts.budget_id", "the account name must be unique for the budget"},
		{http.StatusBadRequest, "UNIQUE constraint failed: categories.name, categories.budget_id", "the category name must be unique for the budget"},
		{http.StatusBadRequest, "UNIQUE constraint failed: envelopes.name, envelopes.category_id", "the envelope name must be unique for the category"},
		{http.StatusBadRequest, "UNIQUE constraint failed: allocations.month, allocations.envelope_id", "you can not create multiple allocations for the same month"},
		{http.StatusBadRequest, "constraint failed: FOREIGN KEY constraint failed", "a resource you are referencing in another resource does not exist"},
		{http.StatusInternalServerError, "This is a very weird error", "an error occurred on the server during your request, please contact your server administrator. The request id is '', send this to your server administrator to help them finding the problem"},
		{http.StatusInternalServerError, "attempt to write a readonly database (1032)", "the database is currently in read-only mode, please try again later"},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			err := errors.New(tt.err)
			status := httperrors.DBError(c, err)
			assert.Equal(t, tt.code, status.Status)
			assert.Equal(t, tt.msg, status.Error())
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		code       int    // The status code that should be set in the httperrors.Error
		err        string // The error string for the httperrors.Error.Err
		parseError error  // The error to parse
	}{
		{http.StatusNotFound, httperrors.ErrNoResource.Error(), gorm.ErrRecordNotFound},
		{http.StatusInternalServerError, "an error occurred on the server during your request", &sqlite.Error{}},
		{http.StatusBadRequest, models.ErrAllocationZero.Error(), models.ErrAllocationZero},
		{http.StatusBadRequest, models.ErrGoalAmountNotPositive.Error(), models.ErrGoalAmountNotPositive},
		{http.StatusInternalServerError, httperrors.ErrDatabaseClosed.Error(), errors.New("sql: database is closed")},
		{http.StatusBadRequest, httperrors.ErrRequestBodyEmpty.Error(), io.EOF},
		{http.StatusBadRequest, "Test Message", &time.ParseError{Message: "Test Message"}},
		{http.StatusInternalServerError, "an error occurred on the server during your request", errors.New("Some random error")},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.code), func(t *testing.T) {
			status := httperrors.Parse(c, tt.parseError)
			assert.Equal(t, tt.code, status.Status)
			assert.Contains(t, status.Err.Error(), tt.err)
		})
	}
}

func TestDatabaseNo(t *testing.T) {
	tests := []struct {
		code int
		err  string
		msg  string
	}{
		{http.StatusBadRequest, "CHECK constraint failed: source_destination_different", "source and destination accounts for a transaction must be different"},
		{http.StatusBadRequest, "UNIQUE constraint failed: categories.name, categories.budget_id", "the category name must be unique for the budget"},
		{http.StatusBadRequest, "UNIQUE constraint failed: envelopes.name, envelopes.category_id", "the envelope name must be unique for the category"},
		{http.StatusBadRequest, "UNIQUE constraint failed: allocations.month, allocations.envelope_id", "you can not create multiple allocations for the same month"},
		{http.StatusBadRequest, "constraint failed: FOREIGN KEY constraint failed", "a resource you are referencing in another resource does not exist"},
		{http.StatusInternalServerError, "This is a very weird error", "an error occurred on the server during your request, please contact your server administrator. The request id is '', send this to your server administrator to help them finding the problem"},
		{http.StatusInternalServerError, "attempt to write a readonly database (1032)", "the database is currently in read-only mode, please try again later"},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			err := errors.New(tt.err)
			status := httperrors.DBError(c, err)
			assert.Equal(t, tt.code, status.Status)
			assert.Equal(t, tt.msg, status.Error())
		})
	}
}

func TestNewPlainString(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.New(c, http.StatusBadRequest, "Non-formatted test message")
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Non-formatted test message", test.DecodeError(t, w.Body.Bytes()))
}

func TestNewFormatString(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		httperrors.New(c, http.StatusBadRequest, "This is a formatting string with parameters that contain %#v and %s", "a string", decimal.NewFromFloat(3.141))
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "This is a formatting string with parameters that contain \"a string\" and 3.141", test.DecodeError(t, w.Body.Bytes()))
}
