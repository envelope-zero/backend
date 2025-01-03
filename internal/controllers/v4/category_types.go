package v4

import (
	"fmt"

	"github.com/envelope-zero/backend/v5/internal/models"
	ez_uuid "github.com/envelope-zero/backend/v5/internal/uuid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CategoryEditable represents all user configurable parameters
type CategoryEditable struct {
	Name     string    `json:"name" example:"Saving" default:""`                             // Name of the category
	BudgetID uuid.UUID `json:"budgetId" example:"52d967d3-33f4-4b04-9ba7-772e5ab9d0ce"`      // ID of the budget the category belongs to
	Note     string    `json:"note" example:"All envelopes for long-term saving" default:""` // Notes about the category
	Archived bool      `json:"archived" example:"true" default:"false"`                      // Is the category archived?
}

func (editable CategoryEditable) model() models.Category {
	return models.Category{
		BudgetID: editable.BudgetID,
		Name:     editable.Name,
		Note:     editable.Note,
		Archived: editable.Archived,
	}
}

type CategoryLinks struct {
	Self      string `json:"self" example:"https://example.com/api/v4/categories/3b1ea324-d438-4419-882a-2fc91d71772f"`              // The category itself
	Envelopes string `json:"envelopes" example:"https://example.com/api/v4/envelopes?category=3b1ea324-d438-4419-882a-2fc91d71772f"` // Envelopes for this category
}

type Category struct {
	models.DefaultModel
	CategoryEditable
	Links CategoryLinks `json:"links"`

	// These fields are computed
	Envelopes []Envelope `json:"envelopes"` // Envelopes for the category
}

func newCategory(c *gin.Context, db *gorm.DB, model models.Category) (Category, error) {
	url := c.GetString(string(models.DBContextURL))

	category := Category{
		DefaultModel: model.DefaultModel,
		CategoryEditable: CategoryEditable{
			BudgetID: model.BudgetID,
			Name:     model.Name,
			Note:     model.Note,
			Archived: model.Archived,
		},
		Links: CategoryLinks{
			Self:      fmt.Sprintf("%s/v4/categories/%s", url, model.ID),
			Envelopes: fmt.Sprintf("%s/v4/envelopes?category=%s", url, model.ID),
		},
	}

	envelopes, err := model.Envelopes(db)
	if err != nil {
		return Category{}, err
	}

	for _, envelope := range envelopes {
		category.Envelopes = append(category.Envelopes, newEnvelope(c, envelope))
	}

	return category, nil
}

type CategoryListResponse struct {
	Data       []Category  `json:"data"`                                                          // List of Categories
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type CategoryCreateResponse struct {
	Data  []CategoryResponse `json:"data"`                                                          // List of the created Categories or their respective error
	Error *string            `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

func (c *CategoryCreateResponse) appendError(err error, currentStatus int) int {
	s := err.Error()
	c.Data = append(c.Data, CategoryResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	newStatus := status(err)
	if newStatus > currentStatus {
		return newStatus
	}

	return currentStatus
}

type CategoryResponse struct {
	Data  *Category `json:"data"`                                                          // Data for the Category
	Error *string   `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type CategoryQueryFilter struct {
	BudgetID ez_uuid.UUID `form:"budget"`                     // By ID of the Budget
	Name     string       `form:"name" filterField:"false"`   // By name
	Note     string       `form:"note" filterField:"false"`   // By note
	Archived bool         `form:"archived"`                   // Is the Category archived?
	Search   string       `form:"search" filterField:"false"` // By string in name or note
	Offset   uint         `form:"offset" filterField:"false"` // The offset of the first Category returned. Defaults to 0.
	Limit    int          `form:"limit" filterField:"false"`  // Maximum number of Categories to return. Defaults to 50.
}

func (f CategoryQueryFilter) model() (models.Category, error) {
	return models.Category{
		BudgetID: f.BudgetID.UUID,
		Archived: f.Archived,
	}, nil
}
