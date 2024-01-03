package router_test

import (
	"net/url"
	"os"
	"testing"

	"github.com/envelope-zero/backend/v4/pkg/router"
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

func TestPprofOff(t *testing.T) {
	os.Setenv("ENABLE_PPROF", "false")
	url, _ := url.Parse("http://example.com")

	r, teardown, err := router.Config(url)
	defer teardown()
	assert.Nil(t, err, "Error on router initialization")

	router.AttachRoutes(r.Group("/"))

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

	_, teardown, err := router.Config(url)
	defer teardown()

	assert.Nil(t, err)
	os.Unsetenv("CORS_ALLOW_ORIGINS")
}
