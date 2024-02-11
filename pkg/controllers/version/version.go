package version

import (
	"net/http"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// Version of the API
//
// This is set at build time, see Makefile.
var apiVersion = "0.0.0"

type Response struct {
	Data Object `json:"data"` // Data object for the version endpoint
}
type Object struct {
	Version string `json:"version" example:"1.1.0"` // the running version of the Envelope Zero backend
}

func RegisterRoutes(r *gin.RouterGroup, version string) {
	// set the API version so that responses are correct
	apiVersion = version

	r.GET("", Get)
	r.OPTIONS("", Options)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			General
// @Success		204
// @Router			/version [options]
func Options(c *gin.Context) {
	httputil.OptionsGet(c)
}

// @Summary		API version
// @Description	Returns the software version of the API
// @Tags			General
// @Success		200	{object}	Response
// @Router			/version [get]
func Get(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Data: Object{
			Version: apiVersion,
		},
	})
}
