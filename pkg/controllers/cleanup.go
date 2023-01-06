package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/gin-gonic/gin"
)

// DeleteAll permanently deletes all resources in the database
//
//	@Summary		Delete everything
//	@Description	Permanently deletes all resources
//	@Tags			v1
//	@Success		204
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1 [delete]
func (co Controller) DeleteAll(c *gin.Context) {
	err := co.DB.Unscoped().Where("true").Delete(&models.Transaction{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = co.DB.Unscoped().Where("true").Delete(&models.Allocation{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = co.DB.Unscoped().Where("true").Delete(&models.Envelope{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = co.DB.Unscoped().Where("true").Delete(&models.Category{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = co.DB.Unscoped().Where("true").Delete(&models.Account{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = co.DB.Unscoped().Where("true").Delete(&models.Budget{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = co.DB.Unscoped().Where("true").Delete(&models.MonthConfig{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
