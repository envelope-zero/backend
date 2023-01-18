package router_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/envelope-zero/backend/v2/pkg/controllers"
	"github.com/envelope-zero/backend/v2/pkg/database"
	"github.com/envelope-zero/backend/v2/pkg/router"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// decodeResponse decodes an HTTP response into a target struct.
func decodeResponse(t *testing.T, r *httptest.ResponseRecorder, target interface{}) {
	err := json.NewDecoder(r.Body).Decode(target)
	if err != nil {
		assert.FailNow(t, "Parsing error", "Unable to parse response from server %q into %v, '%v', Request ID: %s", r.Body, reflect.TypeOf(target), err, r.Result().Header.Get("x-request-id"))
	}
}

func TestGinMode(t *testing.T) {
	os.Setenv("GIN_MODE", "debug")
	url, _ := url.Parse("http://example.com")

	r, err := router.Config(url)
	assert.Nil(t, err, "Error on router initialization")

	db, err := database.Connect(":memory:?_pragma=foreign_keys(1)")
	assert.Nil(t, err, "Error on database connection")

	router.AttachRoutes(controllers.Controller{DB: db}, r.Group("/"))

	assert.Nil(t, err, "%T: %v", err, err)
	assert.True(t, gin.IsDebugging())

	os.Unsetenv("GIN_MODE")
}

func TetsPprofOff(t *testing.T) {
	os.Setenv("ENABLE_PPROF", "false")
	url, _ := url.Parse("http://example.com")

	r, err := router.Config(url)
	assert.Nil(t, err, "Error on router initialization")

	db, err := database.Connect(":memory:?_pragma=foreign_keys(1)")
	assert.Nil(t, err, "Error on database connection")

	router.AttachRoutes(controllers.Controller{DB: db}, r.Group("/"))

	for _, r := range r.Routes() {
		assert.NotContains(t, r.Path, "pprof", "pprof routes are registered erroneously! Route: %s", r)
	}

	os.Unsetenv("ENABLE_PPROF")
}

// TestCorsSetting checks that setting of CORS works.
// It does not check the actual headers as this is already done in testing of the module.
func TestCorsSetting(t *testing.T) {
	os.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:3000 https://example.com")
	url, _ := url.Parse("http://example.com")

	_, err := router.Config(url)

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

	decodeResponse(t, w, &lr)
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

	decodeResponse(t, w, &lr)
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

	decodeResponse(t, w, &lr)
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
