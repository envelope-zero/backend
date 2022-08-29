package router

import (
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func URLMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiURL, ok := os.LookupEnv("API_URL")
		if !ok {
			// If we ever reach this, the check in Router is broken. This is very unlikely.
			log.Error().Str("error", "Environment variable API_URL must be set").Msg("URLMiddleware")
			c.Next()
			return
		}

		url, err := url.Parse(apiURL)
		if err != nil {
			log.Error().Str("error", "Environment variable API_URL must be a valid URL")
			c.Next()
			return
		}

		c.Set("baseURL", url.String())
		c.Next()
	}
}
