package router_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/router"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestURLMiddlewareContextSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	os.Setenv("API_URL", "https://ez.example.com:8081/api")

	r.GET("/envelopes", func(ctx *gin.Context) {
		urlMiddleware := router.URLMiddleware()
		urlMiddleware(c)

		c.JSON(http.StatusOK, map[string]string{
			"baseURL":    c.GetString("baseURL"),
			"requestURL": c.GetString("requestURL"),
		})
	})

	urls := struct {
		BaseURL    string `json:"baseURL"`
		RequestURL string `json:"requestURL"`
	}{}

	// Make and decode repsonse
	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/envelopes", nil)
	r.ServeHTTP(w, c.Request)
	test.DecodeResponse(t, w, &urls)

	assert.Equal(t, "https://ez.example.com:8081/api", urls.BaseURL)
	assert.Equal(t, "https://ez.example.com:8081/api/envelopes", urls.RequestURL)

	os.Unsetenv("API_URL")
}

func TestURLMiddlewareNotAnURL(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	os.Setenv("API_URL", "https://ez.examp\\?le.com:8081/api")

	r.GET("/envelopes", func(ctx *gin.Context) {
		urlMiddleware := router.URLMiddleware()
		urlMiddleware(c)

		c.JSON(http.StatusOK, map[string]string{
			"baseURL":    c.GetString("baseURL"),
			"requestURL": c.GetString("requestURL"),
		})
	})

	urls := struct {
		BaseURL    string `json:"baseURL"`
		RequestURL string `json:"requestURL"`
	}{}

	// Make and decode repsonse
	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/envelopes", nil)
	r.ServeHTTP(w, c.Request)
	test.DecodeResponse(t, w, &urls)

	assert.Equal(t, "", urls.BaseURL)
	assert.Equal(t, "", urls.RequestURL)

	os.Unsetenv("API_URL")
}

func TestURLMiddlewareEnvNotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/envelopes", func(ctx *gin.Context) {
		urlMiddleware := router.URLMiddleware()
		urlMiddleware(c)

		c.JSON(http.StatusOK, map[string]string{
			"baseURL":    c.GetString("baseURL"),
			"requestURL": c.GetString("requestURL"),
		})
	})

	urls := struct {
		BaseURL    string `json:"baseURL"`
		RequestURL string `json:"requestURL"`
	}{}

	// Make and decode repsonse
	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/envelopes", nil)
	r.ServeHTTP(w, c.Request)
	test.DecodeResponse(t, w, &urls)

	assert.Equal(t, "", urls.BaseURL)
	assert.Equal(t, "", urls.RequestURL)
}
