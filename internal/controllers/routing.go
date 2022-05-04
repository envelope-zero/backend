package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/envelope-zero/backend/docs"
	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// This is set at build time, see Makefile.
var version = "0.0.0"

// Router controls the routes for the API.
func Router() (*gin.Engine, error) {
	// gin uses debug as the default mode, we use release for
	// security reasons
	ginMode, ok := os.LookupEnv("GIN_MODE")
	if !ok {
		gin.SetMode("release")
	} else {
		gin.SetMode(ginMode)
	}

	// Log format can be explicitly set.
	// If it is not set, it defaults to human readable for development
	// and JSON for release
	logFormat, ok := os.LookupEnv("LOG_FORMAT")
	output := io.Writer(os.Stdout)
	if (!ok && gin.IsDebugging()) || (ok && logFormat == "human") {
		output = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if gin.IsDebugging() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = log.Output(output).With().Timestamp().Logger()

	// Set up the router and middlewares
	r := gin.New()

	// Don’t process X-Forwarded-For header as we do not do anything with
	// client IPs
	r.ForwardedByClientIP = false

	// Send a HTTP 405 (Method not allowed) for all paths where there is
	// a handler, but not for the specific method used
	r.HandleMethodNotAllowed = true

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

	// CORS settings
	allowOrigins, ok := os.LookupEnv("CORS_ALLOW_ORIGINS")
	if ok {
		log.Debug().Str("allowOrigins", allowOrigins).Msg("CORS")

		r.Use(cors.New(cors.Config{
			AllowOrigins:     strings.Fields(allowOrigins),
			AllowMethods:     []string{"OPTIONS", "GET", "POST", "PATCH", "DELETE"},
			AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
			AllowCredentials: true,
		}))
	}

	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, numHandlers int) {
		log.Debug().Str("method", httpMethod).Str("path", absolutePath).Str("handler", handlerName).Int("handlers", numHandlers).Msg("route")
	}

	// Don’t trust any proxy. We do not process any client IPs,
	// therefore we don’t need to trust anyone here.
	_ = r.SetTrustedProxies([]string{})

	err := models.ConnectDatabase()
	if err != nil {
		return nil, fmt.Errorf("Database connection failed with: %s", err.Error())
	}

	r.GET("", GetRoot)
	r.OPTIONS("", OptionsRoot)
	r.GET("/version", VersionRoot)

	r.OPTIONS("/version", OptionsVersion)

	docs.SwaggerInfo.Title = "Envelope Zero"
	docs.SwaggerInfo.Version = version
	docs.SwaggerInfo.Description = "The backend for Envelope Zero, a zero based envelope budgeting solution. Check out the source code at https://github.com/envelope-zero/backend."

	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 setup
	v1 := r.Group("/v1")
	{
		v1.GET("", GetV1)
		v1.OPTIONS("", OptionsV1)
	}

	budgets := v1.Group("/budgets")
	RegisterBudgetRoutes(budgets)

	return r, nil
}

type RootResponse struct {
	Links RootLinks `json:"links"`
}

type RootLinks struct {
	Docs    string `json:"docs" example:"https://example.com/api/docs/index.html"`
	Version string `json:"version" example:"https://example.com/api/version"`
	V1      string `json:"v1" example:"https://example.com/api/v1"`
}

// @Summary      API root
// @Description  Entrypoint for the API, listing all endpoints
// @Tags         General
// @Success      200  {object}  RootResponse
// @Router       / [get]
func GetRoot(c *gin.Context) {
	url := httputil.RequestHost(c)

	c.JSON(http.StatusOK, RootResponse{
		Links: RootLinks{
			Docs:    url + "/docs/index.html",
			Version: url + "/version",
			V1:      httputil.RequestPathV1(c),
		},
	})
}

type VersionResponse struct {
	Data VersionObject `json:"data"`
}
type VersionObject struct {
	Version string `json:"version" example:"1.1.0"`
}

// @Sumary       API version
// @Description  Returns the software version of the API
// @Tags         General
// @Success      200  {object}  VersionResponse
// @Router       /version [get]
func VersionRoot(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{
		Data: VersionObject{
			Version: version,
		},
	})
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         General
// @Success      204
// @Router       / [options]
func OptionsRoot(c *gin.Context) {
	httputil.OptionsGet(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         General
// @Success      204
// @Router       /version [options]
func OptionsVersion(c *gin.Context) {
	httputil.OptionsGet(c)
}

type V1Response struct {
	Links V1Links `json:"links"`
}

type V1Links struct {
	Budgets string `json:"budgets" example:"https://example.com/api/v1"`
}

// @Sumary       v1 API
// @Description  Returns general information about the v1 API
// @Tags         General
// @Success      200  {object}  V1Response
// @Router       /v1 [get]
func GetV1(c *gin.Context) {
	c.JSON(http.StatusOK, V1Response{
		Links: V1Links{
			Budgets: httputil.RequestPathV1(c) + "/budgets",
		},
	})
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         General
// @Success      204
// @Router       /v1 [options]
func OptionsV1(c *gin.Context) {
	httputil.OptionsGet(c)
}
