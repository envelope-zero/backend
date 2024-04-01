package router

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	docs "github.com/envelope-zero/backend/v5/api"
	"github.com/envelope-zero/backend/v5/pkg/controllers/healthz"
	"github.com/envelope-zero/backend/v5/pkg/controllers/root"
	v4 "github.com/envelope-zero/backend/v5/pkg/controllers/v4"
	version_controller "github.com/envelope-zero/backend/v5/pkg/controllers/version"
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

// Version of the API
//
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
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "This HTTP method is not allowed for the endpoint you called",
		})
	})

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
	gin.DebugPrintRouteFunc = func(_, _, _ string, _ int) {}

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
func AttachRoutes(group *gin.RouterGroup) {
	ezLogger := logger.SetLogger(
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
		}),
	)

	// skipLogger returns the correct logger for the route
	//
	// This is either a logger that skips logging or the ezLogger
	skipLogger := func(path, envVar string) gin.HandlerFunc {
		disable, ok := os.LookupEnv(envVar)
		if ok && disable == "true" {
			return logger.SetLogger(
				logger.WithSkipPath([]string{path}),
			)
		}

		return ezLogger
	}

	// metrics
	group.GET("/metrics", skipLogger("/metrics", "DISABLE_METRICS_LOGS"), gin.WrapH(promhttp.Handler()))

	// healthz
	healthzGroup := group.Group("/healthz", skipLogger("/healthz", "DISABLE_HEALTHZ_LOGS"))
	healthz.RegisterRoutes(healthzGroup.Group(""))

	// All groups that can disable logs are registered, register logger by default
	group.Use(ezLogger)

	// pprof performance profiles
	enablePprof, ok := os.LookupEnv("ENABLE_PPROF")
	if ok && enablePprof == "true" {
		pprof.RouteRegister(group, "debug/pprof")
	}

	// Swagger API docs
	group.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Unversioned global endpoints
	{
		root.RegisterRoutes(group.Group(""))
		version_controller.RegisterRoutes(group.Group("/version"), version)
	}

	// v4
	{
		v4Group := group.Group("/v4")
		v4.RegisterRootRoutes(v4Group.Group(""))
		v4.RegisterAccountRoutes(v4Group.Group("/accounts"))
		v4.RegisterBudgetRoutes(v4Group.Group("/budgets"))
		v4.RegisterCategoryRoutes(v4Group.Group("/categories"))
		v4.RegisterEnvelopeRoutes(v4Group.Group("/envelopes"))
		v4.RegisterExportRoutes(v4Group.Group("/export"), version)
		v4.RegisterGoalRoutes(v4Group.Group("/goals"))
		v4.RegisterImportRoutes(v4Group.Group("/import"))
		v4.RegisterMatchRuleRoutes(v4Group.Group("/match-rules"))
		v4.RegisterMonthConfigRoutes(v4Group.Group("/envelopes"))
		v4.RegisterMonthRoutes(v4Group.Group("/months"))
		v4.RegisterTransactionRoutes(v4Group.Group("/transactions"))
	}
}
