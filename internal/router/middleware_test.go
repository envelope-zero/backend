package router_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/router"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestURLMiddlewareContextSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	os.Setenv("API_URL", "https://ez.example.com:8081/api")

	r.GET("/envelopes", func(ctx *gin.Context) {
		router.URLMiddleware()(c)
		c.String(http.StatusOK, c.GetString("baseURL"))
	})

	// Make and decode repsonse
	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/envelopes", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "https://ez.example.com:8081/api", w.Body.String())

	os.Unsetenv("API_URL")
}

func TestURLMiddlewareNotAnURL(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	os.Setenv("API_URL", "https://ez.examp\\?le.com:8081/api")

	r.GET("/envelopes", func(ctx *gin.Context) {
		router.URLMiddleware()(c)
		c.String(http.StatusOK, c.GetString("baseURL"))
	})

	// Make and decode repsonse
	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/envelopes", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "", w.Body.String())

	os.Unsetenv("API_URL")
}

func TestURLMiddlewareEnvNotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/envelopes", func(ctx *gin.Context) {
		urlMiddleware := router.URLMiddleware()
		urlMiddleware(c)

		c.String(http.StatusOK, c.GetString("baseURL"))
	})

	// Make and decode repsonse
	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/envelopes", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "", w.Body.String())
}
