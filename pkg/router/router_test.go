package router_test

import (
	"net/url"
	"testing"

	"github.com/envelope-zero/backend/v5/pkg/router"
	"github.com/stretchr/testify/assert"
)

func TestGinMode(t *testing.T) {
	url, _ := url.Parse("http://example.com")

	r, teardown, err := router.Config(url)
	defer teardown()

	assert.Nil(t, err, "Error on router initialization")
	router.AttachRoutes(r.Group("/"))
	assert.Nil(t, err, "%T: %v", err, err)
}
