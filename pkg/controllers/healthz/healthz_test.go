package healthz_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v4/pkg/controllers/healthz"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.OPTIONS("/healthz", func(ctx *gin.Context) {
		healthz.Options(c)
	})

	c.Request, _ = http.NewRequest(http.MethodOptions, "http://example.com/healthz", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "OPTIONS, GET", w.Header().Get("allow"))
}

func TestGet(t *testing.T) {
	require.Nil(t, models.Connect(fmt.Sprintf("%s?_pragma=foreign_keys(1)", test.TmpFile(t))))

	t.Parallel()
	recorder := httptest.NewRecorder()
	c, r := gin.CreateTestContext(recorder)

	r.GET("/", func(ctx *gin.Context) {
		healthz.Get(c)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", nil)
	r.ServeHTTP(recorder, c.Request)

	assert.Equal(t, http.StatusOK, recorder.Code)
}
