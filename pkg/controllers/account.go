package controllers

import (
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
)

type AccountQueryFilter struct {
	Name     string `form:"name" filterField:"false"`   // Fuzzy filter for the account name
	Note     string `form:"note" filterField:"false"`   // Fuzzy filter for the note
	BudgetID string `form:"budget"`                     // By budget ID
	OnBudget bool   `form:"onBudget"`                   // Is the account on-budget?
	External bool   `form:"external"`                   // Is the account external?
	Hidden   bool   `form:"hidden"`                     // Is the account hidden?
	Search   string `form:"search" filterField:"false"` // By string in name or note
}

func (f AccountQueryFilter) ToCreate(c *gin.Context) (models.AccountCreate, bool) {
	budgetID, ok := httputil.UUIDFromStringHandleErrors(c, f.BudgetID)
	if !ok {
		return models.AccountCreate{}, false
	}

	return models.AccountCreate{
		BudgetID: budgetID,
		OnBudget: f.OnBudget,
		External: f.External,
		Hidden:   f.Hidden,
	}, true
}
