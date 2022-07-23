package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/router"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGinMode(t *testing.T) {
	os.Setenv("GIN_MODE", "debug")
	_, err := router.Router()

	assert.Nil(t, err, "%T: %v", err, err)
	assert.True(t, gin.IsDebugging())

	os.Unsetenv("GIN_MODE")
}

// TestCorsSetting checks that setting of CORS works.
// It does not check the actual headers as this is already done in testing of the module.
func TestCorsSetting(t *testing.T) {
	os.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:3000 https://example.com")
	_, err := router.Router()

	assert.Nil(t, err)
	os.Unsetenv("CORS_ALLOW_ORIGINS")
}

func TestGetRoot(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		router.GetRoot(c)
	})

	l := router.RootResponse{
		Links: router.RootLinks{
			Docs:    "https://example.com/docs/index.html",
			Version: "https://example.com/version",
			V1:      "https://example.com/v1",
		},
	}

	var lr router.RootResponse

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", nil)
	r.ServeHTTP(w, c.Request)

	test.DecodeResponse(t, w, &lr)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, l, lr)
}

func TestGetV1(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/v1", func(ctx *gin.Context) {
		router.GetV1(c)
	})

	l := router.V1Response{
		Links: router.V1Links{
			Budgets:      "http://example.com/v1/budgets",
			Accounts:     "http://example.com/v1/accounts",
			Transactions: "http://example.com/v1/transactions",
			Categories:   "http://example.com/v1/categories",
			Envelopes:    "http://example.com/v1/envelopes",
			Allocations:  "http://example.com/v1/allocations",
		},
	}

	var lr router.V1Response

	c.Request, _ = http.NewRequest(http.MethodGet, "http://example.com/v1", nil)
	r.ServeHTTP(w, c.Request)

	test.DecodeResponse(t, w, &lr)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, l, lr)
}

func TestGetVersion(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/version", func(ctx *gin.Context) {
		router.GetVersion(c)
	})

	l := router.VersionResponse{
		Data: router.VersionObject{
			Version: "0.0.0",
		},
	}

	var lr router.VersionResponse

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/version", nil)
	r.ServeHTTP(w, c.Request)

	test.DecodeResponse(t, w, &lr)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, l, lr)
}

func TestOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		f    func(*gin.Context)
	}{
		{"/", router.OptionsRoot},
		{"/version", router.OptionsVersion},
		{"/v1", router.OptionsV1},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.OPTIONS(tt.path, func(ctx *gin.Context) {
				tt.f(c)
			})

			url := fmt.Sprintf("http://example.com%s", tt.path)
			c.Request, _ = http.NewRequest(http.MethodOptions, url, nil)
			r.ServeHTTP(w, c.Request)

			assert.Equal(t, http.StatusNoContent, w.Code)
			assert.Equal(t, http.MethodGet, w.Header().Get("allow"))
		})
	}
}
