package router_test

import (
	"net/url"
	"os"
	"testing"

	"github.com/envelope-zero/backend/v7/internal/router"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGinMode(t *testing.T) {
	os.Setenv("GIN_MODE", "debug")
	url, _ := url.Parse("http://example.com")

	r, teardown, err := router.Config(url)
	defer teardown()

	assert.Nil(t, err, "Error on router initialization")

	router.AttachRoutes(r.Group("/"))

	assert.Nil(t, err, "%T: %v", err, err)
	assert.True(t, gin.IsDebugging())

	os.Unsetenv("GIN_MODE")
}

func TestPprofOn(t *testing.T) {
	os.Setenv("ENABLE_PPROF", "true")
	url, _ := url.Parse("http://example.com")

	r, teardown, err := router.Config(url)
	defer teardown()
	assert.Nil(t, err, "Error on router initialization")

	router.AttachRoutes(r.Group("/"))

	var routes []string
	for _, r := range r.Routes() {
		routes = append(routes, r.Path)
	}
	assert.Contains(t, routes, "/debug/pprof/")

	os.Unsetenv("ENABLE_PPROF")
}

func TestPprofOff(t *testing.T) {
	url, _ := url.Parse("http://example.com")

	r, teardown, err := router.Config(url)
	defer teardown()
	assert.Nil(t, err, "Error on router initialization")

	router.AttachRoutes(r.Group("/"))

	for _, r := range r.Routes() {
		assert.NotContains(t, r.Path, "pprof", "pprof routes are registered erroneously! Route: %s", r)
	}
}

// TestCorsSetting checks that setting of CORS works.
// It does not check the actual headers as this is already done in testing of the module.
func TestCorsSetting(t *testing.T) {
	os.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:3000 https://example.com")
	url, _ := url.Parse("http://example.com")

	_, teardown, err := router.Config(url)
	defer teardown()

	assert.Nil(t, err)
	os.Unsetenv("CORS_ALLOW_ORIGINS")
}
