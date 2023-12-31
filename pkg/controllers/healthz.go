package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// RegisterHealthzRoutes registers the routes for the healthz endpoint.
func (co Controller) RegisterHealthzRoutes(r *gin.RouterGroup) {
	r.OPTIONS("", co.OptionsHealthz)
	r.GET("", co.GetHealthz)
}

// OptionsHealthz returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			General
//	@Success		204
//	@Router			/healthz [options]
func (co Controller) OptionsHealthz(c *gin.Context) {
	httputil.OptionsGet(c)
}

type HealthResponse struct {
	Error error `json:"error" example:"The database cannot be accessed"`
}

// GetHealthz returns data about the application health
//
//	@Summary		Get health
//	@Description	Returns the application health and, if not healthy, an error
//	@Tags			General
//	@Produce		json
//	@Success		204
//	@Failure		500	{object} httperrors.HTTPError
//	@Router			/healthz [get]
func (co Controller) GetHealthz(c *gin.Context) {
	sqlDB, err := co.DB.DB()
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = sqlDB.Ping()
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
