package controllers

import (
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
)

// createMatchRule creates a single matchRule after verifying it is a valid matchRule.
func (co Controller) createMatchRule(c *gin.Context, create models.MatchRuleCreate) (models.MatchRule, httperrors.Error) {
	r := models.MatchRule{
		MatchRuleCreate: create,
	}

	// Check that the referenced account exists
	_, err := getResourceByID[models.Account](c, co, r.AccountID)
	if !err.Nil() {
		return r, err
	}

	// Create the resource
	dbErr := co.DB.Create(&r).Error
	if dbErr != nil {
		return models.MatchRule{}, httperrors.GenericDBError[models.MatchRule](r, c, dbErr)
	}

	return r, httperrors.Error{}
}
