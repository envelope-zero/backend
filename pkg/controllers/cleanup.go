package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/pkg/database"
	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
)

// @Summary     Delete everything
// @Description Permanently deletes all resources
// @Tags        v1
// @Success     204
// @Failure     500 {object} httperrors.HTTPError
// @Router      /v1 [delete]
func DeleteAll(c *gin.Context) {
	err := database.DB.Unscoped().Where("true").Delete(&models.Transaction{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = database.DB.Unscoped().Where("true").Delete(&models.Allocation{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = database.DB.Unscoped().Where("true").Delete(&models.Envelope{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = database.DB.Unscoped().Where("true").Delete(&models.Category{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = database.DB.Unscoped().Where("true").Delete(&models.Account{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	err = database.DB.Unscoped().Where("true").Delete(&models.Budget{}).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
