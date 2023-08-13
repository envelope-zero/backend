package httputil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/httputil"
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

	assert.Equal(t, "OPTIONS, GET", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsGetPost(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsGetPost)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "OPTIONS, GET, POST", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsGetPostDelete(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsGetPostDelete)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "OPTIONS, GET, POST, DELETE", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsGetPostPatchDelete(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsGetPostPatchDelete)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "OPTIONS, GET, POST, PATCH, DELETE", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsPost(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsPost)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "OPTIONS, POST", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsGetPatchDelete(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsGetPatchDelete)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "OPTIONS, GET, PATCH, DELETE", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsGetDelete(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsGetDelete)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "OPTIONS, GET, DELETE", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestOptionsDelete(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", httputil.OptionsDelete)

	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = "example.com"
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "OPTIONS, DELETE", w.Header().Get("allow"))
	assert.Equal(t, http.StatusNoContent, w.Code)
}
