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
	co.RegisterMonthRoutes(v1.Group("/months"))
	co.RegisterImportRoutes(v1.Group("/import"))
	co.RegisterMonthConfigRoutes(v1.Group("/month-configs"))

	// API v2 setup
	v2 := group.Group("/v2")
	{
		v2.GET("", GetV2)
		v2.OPTIONS("", OptionsV2)
	}

	co.RegisterAccountRoutesV2(v2.Group("/accounts"))
	co.RegisterTransactionRoutesV2(v2.Group("/transactions"))
	co.RegisterRenameRuleRoutes(v2.Group("/rename-rules"))
	co.RegisterMatchRuleRoutes(v2.Group("/match-rules"))

	// API v3 setup
	v3 := group.Group("/v3")
	{
		v3.GET("", GetV3)
		v3.OPTIONS("", OptionsV3)
	}

	co.RegisterBudgetRoutesV3(v3.Group("/budgets"))
	co.RegisterTransactionRoutesV3(v3.Group("/transactions"))
	co.RegisterMatchRuleRoutesV3(v3.Group("/match-rules"))
	co.RegisterImportRoutesV3(v3.Group("/import"))
}

type RootResponse struct {
	Links RootLinks `json:"links"` // URLs of API endpoints
}

type RootLinks struct {
	Docs    string `json:"docs" example:"https://example.com/api/docs/index.html"` // Swagger API documentation
	Healthz string `json:"healthz" example:"https://example.com/api/healtzh"`      // Healthz endpoint
	Version string `json:"version" example:"https://example.com/api/version"`      // Endpoint returning the version of the backend
	Metrics string `json:"metrics" example:"https://example.com/api/metrics"`      // Endpoint returning Prometheus metrics
	V1      string `json:"v1" example:"https://example.com/api/v1"`                // List endpoint for all v1 endpoints
	V2      string `json:"v2" example:"https://example.com/api/v2"`                // List endpoint for all v2 endpoints
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
			V1:      c.GetString(string(database.ContextURL)) + "/v1",
			V2:      c.GetString(string(database.ContextURL)) + "/v2",
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

type V1Response struct {
	Links V1Links `json:"links"` // Links for the v1 API
}

type V1Links struct {
	Budgets      string `json:"budgets" example:"https://example.com/api/v1/budgets"`           // URL of budget list endpoint
	Accounts     string `json:"accounts" example:"https://example.com/api/v1/accounts"`         // URL of account list endpoint
	Categories   string `json:"categories" example:"https://example.com/api/v1/categories"`     // URL of category list endpoint
	Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions"` // URL of transaction list endpoint
	Envelopes    string `json:"envelopes" example:"https://example.com/api/v1/envelopes"`       // URL of envelope list endpoint
	Allocations  string `json:"allocations" example:"https://example.com/api/v1/allocations"`   // URL of allocation list endpoint
	Months       string `json:"months" example:"https://example.com/api/v1/months"`             // URL of month list endpoint
	Import       string `json:"import" example:"https://example.com/api/v1/import"`             // URL of import list endpoint
}

// GetV1 returns the link list for v1
//
//	@Summary		v1 API
//	@Description	Returns general information about the v1 API
//	@Tags			v1
//	@Success		200	{object}	V1Response
//	@Router			/v1 [get]
func GetV1(c *gin.Context) {
	c.JSON(http.StatusOK, V1Response{
		Links: V1Links{
			Budgets:      c.GetString(string(database.ContextURL)) + "/v1/budgets",
			Accounts:     c.GetString(string(database.ContextURL)) + "/v1/accounts",
			Categories:   c.GetString(string(database.ContextURL)) + "/v1/categories",
			Transactions: c.GetString(string(database.ContextURL)) + "/v1/transactions",
			Envelopes:    c.GetString(string(database.ContextURL)) + "/v1/envelopes",
			Allocations:  c.GetString(string(database.ContextURL)) + "/v1/allocations",
			Months:       c.GetString(string(database.ContextURL)) + "/v1/months",
			Import:       c.GetString(string(database.ContextURL)) + "/v1/import",
		},
	})
}

// OptionsV1 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			v1
//	@Success		204
//	@Router			/v1 [options]
func OptionsV1(c *gin.Context) {
	httputil.OptionsGetDelete(c)
}

type V2Response struct {
	Links V2Links `json:"links"` // Links for the v2 API
}

type V2Links struct {
	Accounts     string `json:"accounts" example:"https://example.com/api/v2/accounts"`         // URL of transaction list endpoint
	Transactions string `json:"transactions" example:"https://example.com/api/v2/transactions"` // URL of transaction list endpoint
	RenameRules  string `json:"rename-rules" example:"https://example.com/api/v2/rename-rules"` // URL of rename-rule list endpoint
	MatchRules   string `json:"match-rules" example:"https://example.com/api/v2/match-rules"`   // URL of match-rule list endpoint
}

// GetV2 returns the link list for v2
//
//	@Summary		v2 API
//	@Description	Returns general information about the v2 API
//	@Tags			v2
//	@Success		200	{object}	V2Response
//	@Router			/v2 [get]
func GetV2(c *gin.Context) {
	c.JSON(http.StatusOK, V2Response{
		Links: V2Links{
			Accounts:     c.GetString(string(database.ContextURL)) + "/v2/accounts",
			Transactions: c.GetString(string(database.ContextURL)) + "/v2/transactions",
			RenameRules:  c.GetString(string(database.ContextURL)) + "/v2/rename-rules",
			MatchRules:   c.GetString(string(database.ContextURL)) + "/v2/match-rules",
		},
	})
}

// OptionsV2 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			v2
//	@Success		204
//	@Router			/v2 [options]
func OptionsV2(c *gin.Context) {
	httputil.OptionsGet(c)
}

type V3Response struct {
	Links V3Links `json:"links"` // Links for the v2 API
}

type V3Links struct {
	Budgets      string `json:"budgets" example:"https://example.com/api/v3/budgets"`           // URL of Budget collection endpoint
	Transactions string `json:"transactions" example:"https://example.com/api/v3/transactions"` // URL of Transaction collection endpoint
	MatchRules   string `json:"matchRules" example:"https://example.com/api/v3/match-rules"`    // URL of Match Rule collection endpoint
	Import       string `json:"import" example:"https://example.com/api/v3/import"`             // URL of import list endpoint
}

// GetV3 returns the link list for v3
//
//	@Summary		v3 API
//	@Description	Returns general information about the v3 API
//	@Tags			v3
//	@Success		200	{object}	V3Response
//	@Router			/v3 [get]
func GetV3(c *gin.Context) {
	c.JSON(http.StatusOK, V3Response{
		Links: V3Links{
			Budgets:      c.GetString(string(database.ContextURL)) + "/v3/budgets",
			Transactions: c.GetString(string(database.ContextURL)) + "/v3/transactions",
			MatchRules:   c.GetString(string(database.ContextURL)) + "/v3/match-rules",
			Import:       c.GetString(string(database.ContextURL)) + "/v3/import",
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
	httputil.OptionsGet(c)
}
