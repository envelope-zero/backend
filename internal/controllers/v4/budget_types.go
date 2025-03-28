package v4

import (
	"fmt"

	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/gin-gonic/gin"
)

type BudgetEditable struct {
	Name     string `json:"name" example:"Morre's Budget" default:""`       // Name of the budget
	Note     string `json:"note" example:"My personal expenses" default:""` // A longer description of the budget
	Currency string `json:"currency" example:"€" default:""`                // The currency for the budget
}

func (editable BudgetEditable) model() models.Budget {
	return models.Budget{
		Name:     editable.Name,
		Note:     editable.Note,
		Currency: editable.Currency,
	}
}

type BudgetLinks struct {
	Self         string `json:"self" example:"https://example.com/api/v4/budgets/550dc009-cea6-4c12-b2a5-03446eb7b7cf"`                      // The budget itself
	Accounts     string `json:"accounts" example:"https://example.com/api/v4/accounts?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`          // Accounts for this budget
	Categories   string `json:"categories" example:"https://example.com/api/v4/categories?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`      // Categories for this budget
	Envelopes    string `json:"envelopes" example:"https://example.com/api/v4/envelopes?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`        // Envelopes for this budget
	Transactions string `json:"transactions" example:"https://example.com/api/v4/transactions?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`  // Transactions for this budget
	Month        string `json:"month" example:"https://example.com/api/v4/months?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf&month=YYYY-MM"` // This uses 'YYYY-MM' for clients to replace with the actual year and month.
	MatchRules   string `json:"matchRules" example:"https://example.com/api/v4/match-rules?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
}

// Budget is the API v4 representation of a Budget.
type Budget struct {
	models.DefaultModel
	BudgetEditable
	Links BudgetLinks `json:"links"`
}

func newBudget(c *gin.Context, model models.Budget) Budget {
	url := c.GetString(string(models.DBContextURL))

	return Budget{
		DefaultModel: model.DefaultModel,
		BudgetEditable: BudgetEditable{
			Name:     model.Name,
			Note:     model.Note,
			Currency: model.Currency,
		},
		Links: BudgetLinks{
			Self:         fmt.Sprintf("%s/v4/budgets/%s", url, model.ID),
			Accounts:     fmt.Sprintf("%s/v4/accounts?budget=%s", url, model.ID),
			Categories:   fmt.Sprintf("%s/v4/categories?budget=%s", url, model.ID),
			Envelopes:    fmt.Sprintf("%s/v4/envelopes?budget=%s", url, model.ID),
			Transactions: fmt.Sprintf("%s/v4/transactions?budget=%s", url, model.ID),
			Month:        fmt.Sprintf("%s/v4/months?budget=%s&month=YYYY-MM", url, model.ID),
			MatchRules:   fmt.Sprintf("%s/v4/match-rules?budget=%s", url, model.ID),
		},
	}
}

type BudgetListResponse struct {
	Data       []Budget    `json:"data"`                                                          // List of budgets
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type BudgetCreateResponse struct {
	Error *string          `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []BudgetResponse `json:"data"`                                                          // List of created Budgets
}

func (b *BudgetCreateResponse) appendError(err error, currentStatus int) int {
	s := err.Error()
	b.Data = append(b.Data, BudgetResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	newStatus := status(err)
	if newStatus > currentStatus {
		return newStatus
	}

	return currentStatus
}

type BudgetResponse struct {
	Data  *Budget `json:"data"`                                                          // Data for the budget
	Error *string `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type BudgetQueryFilter struct {
	Name     string `form:"name" filterField:"false"`   // By name
	Note     string `form:"note" filterField:"false"`   // By note
	Currency string `form:"currency"`                   // By currency
	Search   string `form:"search" filterField:"false"` // By string in name or note
	Offset   uint   `form:"offset" filterField:"false"` // The offset of the first Budget returned. Defaults to 0.
	Limit    int    `form:"limit" filterField:"false"`  // Maximum number of Budgets to return. Defaults to 50.
}

func (f BudgetQueryFilter) model() models.Budget {
	// Does not return string fields since they are filtered by the controller
	return models.Budget{
		Currency: f.Currency,
	}
}
