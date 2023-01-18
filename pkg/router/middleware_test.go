package router_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/envelope-zero/backend/v2/pkg/database"
	"github.com/envelope-zero/backend/v2/pkg/router"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestURLMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	url, _ := url.Parse("https://ez.example.com:8081/api")

	r.GET("/", func(ctx *gin.Context) {
		router.URLMiddleware(url)(c)
		c.String(http.StatusOK, c.GetString(string(database.ContextURL)))
	})

	// Make and decode response
	c.Request, _ = http.NewRequest(http.MethodGet, "https://ez.example.com/", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, "https://ez.example.com:8081/api", w.Body.String())
}
