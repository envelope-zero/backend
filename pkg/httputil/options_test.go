package httputil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestOptionsGet(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsGet)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.MethodGet, w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsPost(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsGetPost)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "GET, POST", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsGetPatchDelete(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsGetPatchDelete)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "GET, PATCH, DELETE", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsDelete(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsDelete)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "DELETE", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}
