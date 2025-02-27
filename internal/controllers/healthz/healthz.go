package healthz

import (
	"net/http"

	"github.com/envelope-zero/backend/v7/internal/httputil"
	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	r.OPTIONS("", Options)
	r.GET("", Get)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			General
// @Success		204
// @Router			/healthz [options]
func Options(c *gin.Context) {
	httputil.OptionsGet(c)
}

// @Summary		Get health
// @Description	Returns the application health and, if not healthy, an error
// @Tags			General
// @Produce		json
// @Success		204
// @Failure		500	{object} map[string]string
// @Router			/healthz [get]
func Get(c *gin.Context) {
	sqlDB, err := models.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	err = sqlDB.Ping()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}
