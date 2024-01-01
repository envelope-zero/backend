package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// parseMonthQuery takes in the context and parses the request
//
// It verifies that the requested budget exists and parses the ID to return
// the budget resource itself.
func (co Controller) parseMonthQuery(c *gin.Context) (types.Month, models.Budget, httperrors.Error) {
	var query struct {
		QueryMonth
		BudgetID string `form:"budget" example:"81b0c9ce-6fd3-4e1e-becc-106055898a2a"`
	}

	if err := c.BindQuery(&query); err != nil {
		return types.Month{}, models.Budget{}, httperrors.Parse(c, err)
	}

	if query.Month.IsZero() {
		return types.Month{}, models.Budget{}, httperrors.Error{
			Status: http.StatusBadRequest,
			Err:    httperrors.ErrMonthNotSetInQuery,
		}
	}

	budgetID, err := uuid.Parse(query.BudgetID)
	if err != nil {
		return types.Month{}, models.Budget{}, httperrors.Parse(c, err)
	}

	budget, e := getResourceByID[models.Budget](c, co, budgetID)
	if !e.Nil() {
		return types.Month{}, models.Budget{}, e
	}

	return types.MonthOf(query.Month), budget, httperrors.Error{}
}
