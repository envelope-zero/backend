package v4

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
)

// @Summary		Delete everything
// @Description	Permanently deletes all resources
// @Tags			v4
// @Success		204
// @Failure		400		{object}	httperrors.HTTPError
// @Failure		500		{object}	httperrors.HTTPError
// @Param			confirm	query		string	false	"Confirmation to delete all resources. Must have the value 'yes-please-delete-everything'"
// @Router			/v4 [delete]
func Cleanup(c *gin.Context) {
	var params struct {
		Confirm string `form:"confirm"`
	}

	err := c.Bind(&params)
	if err != nil || params.Confirm != "yes-please-delete-everything" {
		c.JSON(http.StatusBadRequest, httperrors.HTTPError{
			Error: httperrors.ErrCleanupConfirmation.Error(),
		})
		return
	}

	// The order is important here since there are foreign keys to consider!
	resources := []models.Model{
		models.MatchRule{},
		models.Goal{},
		models.Transaction{},
		models.MonthConfig{},
		models.Envelope{},
		models.Category{},
		models.Account{},
		models.Budget{},
	}

	// Use a transaction so that we can roll back if errors happen
	tx := models.DB.Begin()

	for _, model := range resources {
		err := tx.Unscoped().Where("true").Delete(&model).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, httperrors.HTTPError{
				Error: err.Error(),
			})
			tx.Rollback()
			return
		}
	}

	tx.Commit()
	c.JSON(http.StatusNoContent, nil)
}
