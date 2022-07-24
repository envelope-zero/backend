package router

import (
	"net/url"
	"os"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func URLMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// We can take those two without parsing as Router already ensures they are correct
		hostProto := os.Getenv("API_HOST_PROTOCOL")
		basePath := os.Getenv("API_BASE_PATH")

		log.Debug().Str("basePath", basePath).Str("requestID", requestid.Get(c)).Msg("URLMiddleware")

		baseURL, _ := url.Parse(hostProto + basePath)
		c.Set("baseURL", baseURL.String())
		c.Set("requestURL", baseURL.String()+c.Request.URL.Path)
	}
}
