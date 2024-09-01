package root

import (
	"net/http"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Links Links `json:"links"`
}

type Links struct {
	Docs    string `json:"docs" example:"https://example.com/api/docs/index.html"` // Swagger API documentation
	Healthz string `json:"healthz" example:"https://example.com/api/healtzh"`      // Healthz endpoint
	Version string `json:"version" example:"https://example.com/api/version"`      // Endpoint returning the version of the backend
	Metrics string `json:"metrics" example:"https://example.com/api/metrics"`      // Endpoint returning Prometheus metrics
	V4      string `json:"v4" example:"https://example.com/api/v4"`                // List endpoint for all v4 endpoints
}

func RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", Get)
	r.OPTIONS("", Options)
}

// @Summary		API root
// @Description	Entrypoint for the API, listing all endpoints
// @Tags			General
// @Success		200	{object}	Response
// @Router			/ [get]
func Get(c *gin.Context) {
	url := c.GetString(string(models.DBContextURL))

	c.JSON(http.StatusOK, Response{
		Links: Links{
			Docs:    url + "/docs/index.html",
			Healthz: url + "/healthz",
			Version: url + "/version",
			Metrics: url + "/metrics",
			V4:      url + "/v4",
		},
	})
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			General
// @Success		204
// @Router			/ [options]
func Options(c *gin.Context) {
	httputil.OptionsGet(c)
}
