package router

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	docs "github.com/envelope-zero/backend/v4/api"
	"github.com/envelope-zero/backend/v4/pkg/controllers/healthz"
	"github.com/envelope-zero/backend/v4/pkg/controllers/root"
	v3 "github.com/envelope-zero/backend/v4/pkg/controllers/v3"
	version_controller "github.com/envelope-zero/backend/v4/pkg/controllers/version"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
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
func AttachRoutes(group *gin.RouterGroup) {
	// Register metrics
	group.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// pprof performance profiles
	enablePprof, ok := os.LookupEnv("ENABLE_PPROF")
	if ok && enablePprof == "true" {
		pprof.RouteRegister(group, "debug/pprof")
	}

	// Swagger API docs
	group.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	{
		// Unversioned global endpoints
		root.RegisterRoutes(group.Group(""))
		healthz.RegisterRoutes(group.Group("/healthz"))
		version_controller.RegisterRoutes(group.Group("/version"), version)
	}

	{
		v3Group := group.Group("/v3")
		v3.RegisterRootRoutes(v3Group.Group(""))
		v3.RegisterAccountRoutes(v3Group.Group("/accounts"))
		v3.RegisterBudgetRoutes(v3Group.Group("/budgets"))
		v3.RegisterCategoryRoutes(v3Group.Group("/categories"))
		v3.RegisterEnvelopeRoutes(v3Group.Group("/envelopes"))
		v3.RegisterGoalRoutes(v3Group.Group("/goals"))
		v3.RegisterImportRoutes(v3Group.Group("/import"))
		v3.RegisterMatchRuleRoutes(v3Group.Group("/match-rules"))
		v3.RegisterMonthConfigRoutes(v3Group.Group("/envelopes"))
		v3.RegisterMonthRoutes(v3Group.Group("/months"))
		v3.RegisterTransactionRoutes(v3Group.Group("/transactions"))
	}
}
