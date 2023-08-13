package router

import (
	"net/url"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/gin-gonic/gin"
)

func URLMiddleware(url *url.URL) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(database.ContextURL), url.String())
		c.Next()
	}
}
