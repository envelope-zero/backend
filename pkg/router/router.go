package router

import (
	"net/url"

	"github.com/envelope-zero/backend/v5/internal/router"
	"github.com/gin-gonic/gin"
)

func AttachRoutes(group *gin.RouterGroup) {
	router.AttachRoutes(group)
}

func Config(url *url.URL) (*gin.Engine, func(), error) {
	return router.Config(url)
}
