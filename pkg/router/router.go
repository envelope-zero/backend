package router

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	docs "github.com/envelope-zero/backend/api"
	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/httputil"
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

func Config() (*gin.Engine, error) {
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
	r.Use(URLMiddleware())
	r.Use(logger.SetLogger(
		logger.WithDefaultLevel(zerolog.InfoLevel),
		logger.WithClientErrorLevel(zerolog.InfoLevel),
		logger.WithServerErrorLevel(zerolog.ErrorLevel),
		logger.WithLogger(func(c *gin.Context, logger zerolog.Logger) zerolog.Logger {
			return logger.With().
				Str("request-id", requestid.Get(c)).
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
		log.Debug().Str("CORS Allowed Origins", allowOrigins).Msg("Router")

		r.Use(cors.New(cors.Config{
			AllowOrigins:     strings.Fields(allowOrigins),
			AllowMethods:     []string{"OPTIONS", "GET", "POST", "PATCH", "DELETE"},
			AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
			AllowCredentials: true,
		}))
	}

	// Disable the gin debug route printing as it clutters logs (and test logs)
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, numHandlers int) {}

	// Don’t trust any proxy. We do not process any client IPs,
	// therefore we don’t need to trust anyone here.
	_ = r.SetTrustedProxies([]string{})

	apiURL, ok := os.LookupEnv("API_URL")
	if !ok {
		return nil, errors.New("environment variable API_URL must be set")
	}

	url, err := url.Parse(apiURL)
	if err != nil {
		return nil, errors.New("environment variable API_URL must be a valid URL")
	}

	log.Debug().Str("API Base URL", url.String()).Str("Host", url.Host).Str("Path", url.Path).Msg("Router")

	docs.SwaggerInfo.Host = url.Host
	docs.SwaggerInfo.BasePath = url.Path
	docs.SwaggerInfo.Title = "Envelope Zero"
	docs.SwaggerInfo.Version = version
	docs.SwaggerInfo.Description = "The backend for Envelope Zero, a zero based envelope budgeting solution. Check out the source code at https://github.com/envelope-zero/backend."

	return r, nil
}

// AttachRoutes attaches the API routes to the router group that is passed in
// Separating this from RouterConfig() allows us to attach it to different
// paths for different use cases, e.g. the standalone version.
func AttachRoutes(co controllers.Controller, group *gin.RouterGroup) {
	group.GET("", GetRoot)
	group.OPTIONS("", OptionsRoot)
	group.GET("/version", GetVersion)

	group.OPTIONS("/version", OptionsVersion)

	group.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 setup
	v1 := group.Group("/v1")
	{
		v1.GET("", GetV1)
		v1.DELETE("", co.DeleteAll)
		v1.OPTIONS("", OptionsV1)
	}

	co.RegisterBudgetRoutes(v1.Group("/budgets"))
	co.RegisterAccountRoutes(v1.Group("/accounts"))
	co.RegisterTransactionRoutes(v1.Group("/transactions"))
	co.RegisterCategoryRoutes(v1.Group("/categories"))
	co.RegisterEnvelopeRoutes(v1.Group("/envelopes"))
	co.RegisterAllocationRoutes(v1.Group("/allocations"))
}

type RootResponse struct {
	Links RootLinks `json:"links"`
}

type RootLinks struct {
	Docs    string `json:"docs" example:"https://example.com/api/docs/index.html"`
	Version string `json:"version" example:"https://example.com/api/version"`
	V1      string `json:"v1" example:"https://example.com/api/v1"`
}

// @Summary     API root
// @Description Entrypoint for the API, listing all endpoints
// @Tags        General
// @Success     200 {object} RootResponse
// @Router      / [get]
func GetRoot(c *gin.Context) {
	c.JSON(http.StatusOK, RootResponse{
		Links: RootLinks{
			Docs:    c.GetString("baseURL") + "/docs/index.html",
			Version: c.GetString("baseURL") + "/version",
			V1:      c.GetString("baseURL") + "/v1",
		},
	})
}

type VersionResponse struct {
	Data VersionObject `json:"data"`
}
type VersionObject struct {
	Version string `json:"version" example:"1.1.0"`
}

// @Summary     API version
// @Description Returns the software version of the API
// @Tags        General
// @Success     200 {object} VersionResponse
// @Router      /version [get]
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{
		Data: VersionObject{
			Version: version,
		},
	})
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        General
// @Success     204
// @Router      / [options]
func OptionsRoot(c *gin.Context) {
	httputil.OptionsGet(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        General
// @Success     204
// @Router      /version [options]
func OptionsVersion(c *gin.Context) {
	httputil.OptionsGet(c)
}

type V1Response struct {
	Links V1Links `json:"links"`
}

type V1Links struct {
	Budgets      string `json:"budgets" example:"https://example.com/api/v1/budgets"`
	Accounts     string `json:"accounts" example:"https://example.com/api/v1/accounts"`
	Categories   string `json:"categories" example:"https://example.com/api/v1/categories"`
	Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions"`
	Envelopes    string `json:"envelopes" example:"https://example.com/api/v1/envelopes"`
	Allocations  string `json:"allocations" example:"https://example.com/api/v1/allocations"`
}

// @Summary     v1 API
// @Description Returns general information about the v1 API
// @Tags        v1
// @Success     200 {object} V1Response
// @Router      /v1 [get]
func GetV1(c *gin.Context) {
	c.JSON(http.StatusOK, V1Response{
		Links: V1Links{
			Budgets:      c.GetString("baseURL") + "/v1/budgets",
			Accounts:     c.GetString("baseURL") + "/v1/accounts",
			Categories:   c.GetString("baseURL") + "/v1/categories",
			Transactions: c.GetString("baseURL") + "/v1/transactions",
			Envelopes:    c.GetString("baseURL") + "/v1/envelopes",
			Allocations:  c.GetString("baseURL") + "/v1/allocations",
		},
	})
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        v1
// @Success     204
// @Router      /v1 [options]
func OptionsV1(c *gin.Context) {
	httputil.OptionsGetDelete(c)
}
