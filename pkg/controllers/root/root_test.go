package root_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v5/pkg/controllers/root"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.OPTIONS("/", func(_ *gin.Context) {
		root.Options(c)
	})

	c.Request, _ = http.NewRequest(http.MethodOptions, "http://example.com/", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "OPTIONS, GET", w.Header().Get("allow"))
}

func TestGet(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	c, r := gin.CreateTestContext(recorder)

	r.GET("/", func(_ *gin.Context) {
		root.Get(c)
	})

	// Test contexts cannot be injected any middleware, therefore
	// this only tests the path, not the host
	expectedResponse := root.Response{
		Links: root.Links{
			Docs:    "/docs/index.html",
			Healthz: "/healthz",
			Version: "/version",
			Metrics: "/metrics",
			V3:      "/v3",
		},
	}

	var response root.Response

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", nil)
	r.ServeHTTP(recorder, c.Request)

	test.DecodeResponse(t, recorder, &response)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, expectedResponse, response)
}
