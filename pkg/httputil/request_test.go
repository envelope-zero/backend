package httputil_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBindData(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		_ = httputil.BindData(c, &o)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "http://example.com/", bytes.NewBuffer([]byte(`{ "name": "Drink more water!" }`)))
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code, "Binding failed: %s", w.Body.String())
}

func TestBindBrokenData(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		_ = httputil.BindData(c, &o)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte(`{ broken json: "Drink more water!" }`)))
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Binding failed: %s", w.Body.String())
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "the body of your request contains invalid or un-parseable data")
}

func TestBindEmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		_ = httputil.BindData(c, &o)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte("")))
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Binding failed: %s", w.Body.String())
	assert.Contains(t, test.DecodeError(t, w.Body.Bytes()), "request body must not be empty")
}

func TestUUIDFromString(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		var o struct {
			UUID string `form:"id"`
		}

		_ = c.Bind(&o)
		_, err := httputil.UUIDFromString(c, o.UUID)
		if err != nil {
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
		_, err := httputil.UUIDFromString(c, o.UUID)
		if err != nil {
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
		_, err := httputil.UUIDFromString(c, o.UUID)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
		}
		c.Status(http.StatusOK)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/?id=", bytes.NewBuffer([]byte("")))
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusOK, w.Code)
}
