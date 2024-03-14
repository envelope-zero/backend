package v4

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
)

var backendVersion string

func RegisterExportRoutes(r *gin.RouterGroup, version string) {
	backendVersion = version

	{
		r.OPTIONS("", OptionsExport)
		r.GET("", GetExport)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Export
// @Success		204
// @Router			/v4/export [options]
func OptionsExport(c *gin.Context) {
	httputil.OptionsGet(c)
}

// @Summary		Export
// @Description	Exports all resources for the instance
// @Tags			Export
// @Produce		json
// @Success		200	{object}	ExportResponse
// @Failure		500	{object}	ExportResponse
// @Router			/v4/export [get]
func GetExport(c *gin.Context) {
	resources := make(map[string]json.RawMessage)

	for _, model := range models.Registry {
		b, err := model.Export()
		if err != nil {
			c.JSON(status(err), httpError{
				Error: err.Error(),
			})
			return
		}

		resources[reflect.TypeOf(model).Name()] = b
	}

	c.JSON(http.StatusOK, ExportResponse{
		Version:      backendVersion,
		Data:         resources,
		CreationTime: time.Now(),
		Clacks:       "GNU Terry Pratchett",
	})
}
