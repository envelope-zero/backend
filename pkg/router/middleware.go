package router

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
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
