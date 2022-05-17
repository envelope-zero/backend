package httputil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestHostNaked(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		c.String(http.StatusOK, httputil.RequestHost(c))
	})

	// Check without reverse proxy headers
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, "http://example.com", w.Body.String())
}

func TestRequestHostReverseProxy(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		c.String(http.StatusOK, httputil.RequestHost(c))
	})

	// Check with reverse proxy, but without x-forwarded-prefix
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "::1"
	c.Request.Header.Set("x-forwarded-host", "example.com")
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, "http://example.com/api", w.Body.String())
}

func TestRequestHostPrefix(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		c.String(http.StatusOK, httputil.RequestHost(c))
	})

	// Check with x-forwarded-prefix
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "::1"
	c.Request.Header.Set("x-forwarded-host", "example.com")
	c.Request.Header.Set("x-forwarded-prefix", "/backend")
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, "http://example.com/backend", w.Body.String())
}
