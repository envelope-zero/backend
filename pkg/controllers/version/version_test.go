package version_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v4/pkg/controllers/version"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.OPTIONS("/version", func(ctx *gin.Context) {
		version.Options(c)
	})

	c.Request, _ = http.NewRequest(http.MethodOptions, "http://example.com/version", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "OPTIONS, GET", w.Header().Get("allow"))
}

func TestGetVersion(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	c, r := gin.CreateTestContext(recorder)

	r.GET("/version", func(ctx *gin.Context) {
		version.Get(c)
	})

	expectedResponse := version.Response{
		Data: version.Object{
			Version: "0.0.0",
		},
	}

	var response version.Response

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/version", nil)
	r.ServeHTTP(recorder, c.Request)

	test.DecodeResponse(t, recorder, &response)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, expectedResponse, response)
}
