package httputil_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBindDataHandleErrors(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		_ = httputil.BindDataHandleErrors(c, &o)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "http://example.com/", bytes.NewBuffer([]byte(`{ "name": "Drink more water!" }`)))
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code, "Binding failed: %s", w.Body.String())
}

func TestBindDataHandleErrorsBrokenData(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		_ = httputil.BindDataHandleErrors(c, &o)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte(`{ broken json: "Drink more water!" }`)))
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Binding failed: %s", w.Body.String())
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "the body of your request contains invalid or un-parseable data")
}

func TestBindHandleErrorsEmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		_ = httputil.BindDataHandleErrors(c, &o)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte("")))
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Binding failed: %s", w.Body.String())
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "request body must not be empty")
}

// TestBindData verifies that BindData succeeds on valid data.
func TestBindData(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		err := httputil.BindData(c, &o)
		assert.True(t, err.Nil())
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte(`{ "name": "Drink more water!" }`)))
	r.ServeHTTP(w, c.Request)
}

// TestBindDataInvalidBody verifies that BindData returns the correct error on an invalid body.
func TestBindDataInvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		err := httputil.BindData(c, &o)
		assert.Equal(t, httperrors.Error{Status: http.StatusBadRequest, Err: httperrors.ErrInvalidBody}, err)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte(`{ invalid json: "Drink more water! }`)))
	r.ServeHTTP(w, c.Request)
}

// TestBindDataEmptyBody verifies that BindData returns the correct error on an empty body.
func TestBindDataEmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		err := httputil.BindData(c, &o)
		assert.Equal(t, httperrors.Error{Status: http.StatusBadRequest, Err: httperrors.ErrRequestBodyEmpty}, err)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte("")))
	r.ServeHTTP(w, c.Request)
}

func TestUUIDFromString(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			UUID string `form:"id"`
		}

		_ = c.Bind(&o)
		_, ok := httputil.UUIDFromStringHandleErrors(c, o.UUID)
		if !ok {
			c.AbortWithStatus(http.StatusBadRequest)
		}
		c.Status(http.StatusOK)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/?id=4e743e94-6a4b-44d6-aba5-d77c82103fa7", bytes.NewBuffer([]byte("")))
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUUIDFromStringInvalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			UUID string `form:"id"`
		}

		_ = c.Bind(&o)
		_, ok := httputil.UUIDFromStringHandleErrors(c, o.UUID)
		if !ok {
			c.AbortWithStatus(http.StatusBadRequest)
		}
		c.Status(http.StatusOK)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/?id=not-a-valid-uuid", bytes.NewBuffer([]byte("")))
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUUIDFromStringEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			UUID string `form:"id"`
		}

		_ = c.Bind(&o)
		_, ok := httputil.UUIDFromStringHandleErrors(c, o.UUID)
		if !ok {
			c.AbortWithStatus(http.StatusBadRequest)
		}
		c.Status(http.StatusOK)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/?id=", bytes.NewBuffer([]byte("")))
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusOK, w.Code)
}
