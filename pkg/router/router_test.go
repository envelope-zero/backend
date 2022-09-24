package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/database"
	"github.com/envelope-zero/backend/pkg/router"
	"github.com/envelope-zero/backend/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGinMode(t *testing.T) {
	os.Setenv("GIN_MODE", "debug")
	os.Setenv("API_URL", "http://example.com")

	r, err := router.Config()
	assert.Nil(t, err, "Error on router initialization")

	db, err := database.Connect(":memory:?_pragma=foreign_keys(1)")
	assert.Nil(t, err, "Error on database connection")

	router.AttachRoutes(controllers.Controller{DB: db}, r.Group("/"))

	assert.Nil(t, err, "%T: %v", err, err)
	assert.True(t, gin.IsDebugging())

	os.Unsetenv("GIN_MODE")
	os.Unsetenv("API_URL")
}

func TestEnvUnset(t *testing.T) {
	_, err := router.Config()

	assert.NotNil(t, err, "API_URL is unset, this must lead to an error")
}

func TestEnvNoURL(t *testing.T) {
	os.Setenv("API_URL", "\\:veryMuchNotAURL")
	_, err := router.Config()

	assert.NotNil(t, err, "API_URL is not an URL, this must lead to an error")
}

// TestCorsSetting checks that setting of CORS works.
// It does not check the actual headers as this is already done in testing of the module.
func TestCorsSetting(t *testing.T) {
	os.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:3000 https://example.com")
	os.Setenv("API_URL", "http://example.com")

	_, err := router.Config()

	assert.Nil(t, err)
	os.Unsetenv("CORS_ALLOW_ORIGINS")
	os.Unsetenv("API_URL")
}

func TestGetRoot(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(ctx *gin.Context) {
		router.GetRoot(c)
	})

	// Test contexts cannot be injected any middleware, therefore
	// this only tests the path, not the host
	l := router.RootResponse{
		Links: router.RootLinks{
			Docs:    "/docs/index.html",
			Version: "/version",
			V1:      "/v1",
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

	// Test contexts cannot be injected any middleware, therefore
	// this only tests the path, not the host
	l := router.V1Response{
		Links: router.V1Links{
			Budgets:      "/v1/budgets",
			Accounts:     "/v1/accounts",
			Transactions: "/v1/transactions",
			Categories:   "/v1/categories",
			Envelopes:    "/v1/envelopes",
			Allocations:  "/v1/allocations",
			Months:       "/v1/months",
			Import:       "/v1/import",
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
		path     string
		f        func(*gin.Context)
		expected string
	}{
		{"/", router.OptionsRoot, "OPTIONS, GET"},
		{"/version", router.OptionsVersion, "OPTIONS, GET"},
		{"/v1", router.OptionsV1, "OPTIONS, GET, DELETE"},
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
			assert.Equal(t, tt.expected, w.Header().Get("allow"))
		})
	}
}
