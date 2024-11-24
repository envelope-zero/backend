package router

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v5/internal/httperror"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

func URLMiddleware(url *url.URL) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(models.DBContextURL), url.String())
		c.Next()
	}
}

var metrics = []prometheus.Collector{
	requestCount,
	requestDuration,
}

// registerPrometheusMetrics registers all Prometheus metrics
// with the default registry.
func registerPrometheusMetrics() error {
	for _, c := range metrics {
		if err := prometheus.Register(c); err != nil {
			return fmt.Errorf("could not register %s with Prometheus", c)
		}
	}

	return nil
}

// unregisterPrometheusMetrics unregisters all Prometheus metrics.
//
// This is needed to cleanly exit.
func unregisterPrometheusMetrics() bool {
	for _, c := range metrics {
		if ok := prometheus.Unregister(c); !ok {
			return false
		}
	}

	return true
}

var requestCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "requests_total",
		Help: "How many HTTP requests processed, partitioned by status code and HTTP method.",
	},
	[]string{"code", "method", "url"},
)

var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "request_duration_seconds",
		Help: "The HTTP request latencies in seconds.",
	},
	[]string{"code", "method", "url"},
)

// MetricsMiddleware updates Prometheus metrics.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Second)

		// Replace all URL parameters with their name to reduce cardinality
		// https://prometheus.io/docs/practices/naming/#labels
		url := c.Request.URL.Path
		for _, p := range c.Params {
			url = strings.Replace(url, p.Value, fmt.Sprintf(":%s", p.Key), 1)
		}

		requestDuration.WithLabelValues(status, c.Request.Method, url).Observe(elapsed)
		requestCount.WithLabelValues(status, c.Request.Method, url).Inc()
	}
}

// TODO: This needs to be cleaned up
func ValidationErrorToText(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "max":
		return fmt.Sprintf("%s cannot be longer than %s", e.Field(), e.Param())
	case "min":
		return fmt.Sprintf("%s must be longer than %s", e.Field(), e.Param())
	case "email":
		return "Invalid email format"
	case "len":
		return fmt.Sprintf("%s must be %s characters long", e.Field(), e.Param())
	}
	return fmt.Sprintf("%s is not valid", e.Field())
}

func ErrorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only run if there are some errors to handle
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				// Find out what type of error it is
				switch e.Type {
				case gin.ErrorTypePublic:
					// Only output public errors if nothing has been written yet
					if !c.Writer.Written() {
						c.JSON(c.Writer.Status(), httperror.New(e))
					}

				case gin.ErrorTypeBind:

					// TODO: Check with errors.As first, fall back to just use the error, see https://stackoverflow.com/a/73352006/6733572
					errs := e.Err.(validator.ValidationErrors)
					list := make(map[string]string)
					for _, err := range errs {
						// TODO: Join into a string instead of creating a list
						list[err.Field()] = ValidationErrorToText(err)
					}

					// Make sure we maintain the preset response status
					status := http.StatusBadRequest
					if c.Writer.Status() != http.StatusOK {
						status = c.Writer.Status()
					}
					c.JSON(status, httperror.New(e))
				default:
					requestID := requestid.Get(c)
					log.Error().Str("request-id", requestID).Msgf("%T: %v", e, e.Err)
				}
			}

			// If there was no public or bind error, display default 500 message
			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, httperror.NewFromString("oops, something went wrong"))
			}
		}
	}
}

/*

r.POST("/login", gin.Bind(LoginStruct{}), LoginHandler)

(...)

func  LoginHandler(c *gin.Context) {
    var player *PlayerStruct
    login := c.MustGet(gin.BindKey).(*LoginStruct)
}

*/
