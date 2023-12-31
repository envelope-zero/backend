package router

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	docs "github.com/envelope-zero/backend/v3/api"
	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// This is set at build time, see Makefile.
var version = "0.0.0"

// Config sets up the router, returns a teardown function
// and an error.
func Config(url *url.URL) (*gin.Engine, func(), error) {
	// Set up prometheus metrics
	if err := registerPrometheusMetrics(); err != nil {
		return gin.New(), func() {}, err
	}

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
	r.Use(URLMiddleware(url))
	r.Use(MetricsMiddleware())
	r.NoMethod(func(c *gin.Context) {
		httperrors.New(c, http.StatusMethodNotAllowed, "This HTTP method is not allowed for the endpoint you called")
	})
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

	log.Debug().Str("API Base URL", url.String()).Str("Host", url.Host).Str("Path", url.Path).Msg("Router")
	log.Info().Str("version", version).Msg("Router")

	docs.SwaggerInfo.Host = url.Host
	docs.SwaggerInfo.BasePath = url.Path
	docs.SwaggerInfo.Title = "Envelope Zero"
	docs.SwaggerInfo.Version = version
	docs.SwaggerInfo.Description = "The backend for Envelope Zero, a zero based envelope budgeting solution. Check out the source code at https://github.com/envelope-zero/backend."

	return r, func() { unregisterPrometheusMetrics() }, nil
}

// AttachRoutes attaches the API routes to the router group that is passed in
// Separating this from RouterConfig() allows us to attach it to different
// paths for different use cases, e.g. the standalone version.
func AttachRoutes(co controllers.Controller, group *gin.RouterGroup) {
	group.GET("", GetRoot)
	group.OPTIONS("", OptionsRoot)
	group.GET("/version", GetVersion)
	group.OPTIONS("/version", OptionsVersion)

	// Register metrics
	group.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// pprof performance profiles
	enablePprof, ok := os.LookupEnv("ENABLE_PPROF")
	if ok && enablePprof == "true" {
		pprof.RouteRegister(group, "debug/pprof")
	}

	group.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	co.RegisterHealthzRoutes(group.Group("/healthz"))

	// API v3 setup
	v3 := group.Group("/v3")
	{
		v3.GET("", GetV3)
		v3.DELETE("", co.CleanupV3)
		v3.OPTIONS("", OptionsV3)
	}

	co.RegisterAccountRoutesV3(v3.Group("/accounts"))
	co.RegisterBudgetRoutesV3(v3.Group("/budgets"))
	co.RegisterCategoryRoutesV3(v3.Group("/categories"))
	co.RegisterEnvelopeRoutesV3(v3.Group("/envelopes"))
	co.RegisterGoalRoutesV3(v3.Group("/goals"))
	co.RegisterImportRoutesV3(v3.Group("/import"))
	co.RegisterMatchRuleRoutesV3(v3.Group("/match-rules"))
	co.RegisterMonthConfigRoutesV3(v3.Group("/envelopes"))
	co.RegisterMonthRoutesV3(v3.Group("/months"))
	co.RegisterTransactionRoutesV3(v3.Group("/transactions"))
}

type RootResponse struct {
	Links RootLinks `json:"links"` // URLs of API endpoints
}

type RootLinks struct {
	Docs    string `json:"docs" example:"https://example.com/api/docs/index.html"` // Swagger API documentation
	Healthz string `json:"healthz" example:"https://example.com/api/healtzh"`      // Healthz endpoint
	Version string `json:"version" example:"https://example.com/api/version"`      // Endpoint returning the version of the backend
	Metrics string `json:"metrics" example:"https://example.com/api/metrics"`      // Endpoint returning Prometheus metrics
	V3      string `json:"v3" example:"https://example.com/api/v3"`                // List endpoint for all v3 endpoints
}

// GetRoot returns the link list for the API root
//
//	@Summary		API root
//	@Description	Entrypoint for the API, listing all endpoints
//	@Tags			General
//	@Success		200	{object}	RootResponse
//	@Router			/ [get]
func GetRoot(c *gin.Context) {
	c.JSON(http.StatusOK, RootResponse{
		Links: RootLinks{
			Docs:    c.GetString(string(database.ContextURL)) + "/docs/index.html",
			Healthz: c.GetString(string(database.ContextURL)) + "/healthz",
			Version: c.GetString(string(database.ContextURL)) + "/version",
			Metrics: c.GetString(string(database.ContextURL)) + "/metrics",
			V3:      c.GetString(string(database.ContextURL)) + "/v3",
		},
	})
}

type VersionResponse struct {
	Data VersionObject `json:"data"` // Data object for the version endpoint
}
type VersionObject struct {
	Version string `json:"version" example:"1.1.0"` // the running version of the Envelope Zero backend
}

// GetVersion returns the API version object
//
//	@Summary		API version
//	@Description	Returns the software version of the API
//	@Tags			General
//	@Success		200	{object}	VersionResponse
//	@Router			/version [get]
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{
		Data: VersionObject{
			Version: version,
		},
	})
}

// OptionsRoot returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			General
//	@Success		204
//	@Router			/ [options]
func OptionsRoot(c *gin.Context) {
	httputil.OptionsGet(c)
}

// OptionsVersion returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			General
//	@Success		204
//	@Router			/version [options]
func OptionsVersion(c *gin.Context) {
	httputil.OptionsGet(c)
}

type V3Response struct {
	Links V3Links `json:"links"` // Links for the v3 API
}

type V3Links struct {
	Accounts     string `json:"accounts" example:"https://example.com/api/v3/accounts"`         // URL of Account collection endpoint
	Budgets      string `json:"budgets" example:"https://example.com/api/v3/budgets"`           // URL of Budget collection endpoint
	Categories   string `json:"categories" example:"https://example.com/api/v3/categories"`     // URL of Category collection endpoint
	Envelopes    string `json:"envelopes" example:"https://example.com/api/v3/envelopes"`       // URL of Envelope collection endpoint
	Goals        string `json:"goals" example:"https://example.com/api/v3/goals"`               // URL of goal collection endpoint
	Import       string `json:"import" example:"https://example.com/api/v3/import"`             // URL of import list endpoint
	MatchRules   string `json:"matchRules" example:"https://example.com/api/v3/match-rules"`    // URL of Match Rule collection endpoint
	Months       string `json:"months" example:"https://example.com/api/v3/months"`             // URL of Month endpoint
	Transactions string `json:"transactions" example:"https://example.com/api/v3/transactions"` // URL of Transaction collection endpoint
}

// GetV3 returns the link list for v3
//
//	@Summary		v3 API
//	@Description	Returns general information about the v3 API
//	@Tags			v3
//	@Success		200	{object}	V3Response
//	@Router			/v3 [get]
func GetV3(c *gin.Context) {
	url := c.GetString(string(database.ContextURL))

	c.JSON(http.StatusOK, V3Response{
		Links: V3Links{
			Accounts:     url + "/v3/accounts",
			Budgets:      url + "/v3/budgets",
			Categories:   url + "/v3/categories",
			Envelopes:    url + "/v3/envelopes",
			Goals:        url + "/v3/goals",
			Import:       url + "/v3/import",
			MatchRules:   url + "/v3/match-rules",
			Months:       url + "/v3/months",
			Transactions: url + "/v3/transactions",
		},
	})
}

// OptionsV3 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			v3
//	@Success		204
//	@Router			/v3 [options]
func OptionsV3(c *gin.Context) {
	httputil.OptionsGetDelete(c)
}
