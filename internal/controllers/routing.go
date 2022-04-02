package controllers

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// This is set at build time, see Makefile.
var version = "0.0.0"

// Router controls the routes for the API.
func Router() (*gin.Engine, error) {
	// Set up the router and middlewares
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestid.New())
	r.Use(logger.SetLogger(
		logger.WithDefaultLevel(zerolog.InfoLevel),
		logger.WithClientErrorLevel(zerolog.InfoLevel),
		logger.WithServerErrorLevel(zerolog.ErrorLevel),
		logger.WithLogger(func(c *gin.Context, out io.Writer, latency time.Duration) zerolog.Logger {
			return log.Logger.With().
				Str("request-id", requestid.Get(c)).
				Dur("latency", latency).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Int("status", c.Writer.Status()).
				Int("size", c.Writer.Size()).
				Str("user-agent", c.Request.UserAgent()).
				Logger()
		})))

	err := models.ConnectDatabase()
	if err != nil {
		return nil, fmt.Errorf("Database connection failed with: %s", err.Error())
	}

	// The root path lists the available versions
	r.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"links": map[string]string{
				"v1":      requestURL(c) + "v1",
				"version": requestURL(c) + "version",
			},
		})
	})

	// Options lists the allowed HTTP verbs
	r.OPTIONS("", func(c *gin.Context) {
		c.Header("allow", "GET")
	})

	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]string{
				"version": version,
			},
		})
	})

	r.OPTIONS("/version", func(c *gin.Context) {
		c.Header("allow", "GET")
	})

	// API v1 setup
	v1 := r.Group("/v1")
	{
		v1.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"links": map[string]string{
					"budgets": requestURL(c) + "/budgets",
				},
			})
		})

		v1.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET")
		})
	}

	budgets := v1.Group("/budgets")
	RegisterBudgetRoutes(budgets)

	return r, nil
}
