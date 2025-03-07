package v4

import (
	"github.com/envelope-zero/backend/v7/internal/httputil"
	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/gin-gonic/gin"
)

// resourceOptionsDetail returns the appropriate response for an HTTP OPTIONS request for a specific resource.
//
// Note: This function only works for resources with an ID, not for configurations (like /month-configs) or calculated endpoints (like /months)
func resourceOptionsDetail[R models.Account | models.Budget | models.Category | models.Envelope | models.Goal | models.MatchRule | models.Transaction](c *gin.Context, resource R) {
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
