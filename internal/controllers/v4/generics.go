package v4

import (
	"net/http"

	"github.com/envelope-zero/backend/v7/internal/httputil"
	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/gin-gonic/gin"
)

type Resource interface {
	models.Account | models.Budget | models.Category | models.Envelope | models.Goal | models.MatchRule | models.Transaction
}

// resourceOptionsDetail returns the appropriate response for an HTTP OPTIONS request for a specific resource.
//
// Note: This function only works for resources with an ID, not for configurations (like /month-configs) or calculated endpoints (like /months)
func resourceOptionsDetail[R Resource](c *gin.Context, resource R) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.First(&resource, uri.ID).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

func deleteResource[R Resource](c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var resource R
	err = models.DB.First(&resource, uri.ID).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.Delete(&resource).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
