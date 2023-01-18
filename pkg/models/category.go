package models

import (
	"fmt"

	"github.com/envelope-zero/backend/v2/pkg/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category represents a category of envelopes.
type Category struct {
	DefaultModel
	CategoryCreate
	Budget Budget `json:"-"` // The budget the category belongs to
	Links  struct {
		Self      string `json:"self" example:"https://example.com/api/v1/categories/3b1ea324-d438-4419-882a-2fc91d71772f"`
		Envelopes string `json:"envelopes" example:"https://example.com/api/v1/envelopes?category=3b1ea324-d438-4419-882a-2fc91d71772f"`
	} `json:"links" gorm:"-"`
}

type CategoryCreate struct {
	Name     string    `json:"name" gorm:"uniqueIndex:category_budget_name" example:"Saving" default:""`                        // Name of the category
	BudgetID uuid.UUID `json:"budgetId" gorm:"uniqueIndex:category_budget_name" example:"52d967d3-33f4-4b04-9ba7-772e5ab9d0ce"` // ID of the budget the category belongs to
	Note     string    `json:"note" example:"All envelopes for long-term saving" default:""`                                    // Notes about the category
	Hidden   bool      `json:"hidden" example:"true" default:"false"`                                                           // Is the category hidden?
}

func (c *Category) AfterFind(tx *gorm.DB) (err error) {
	c.links(tx)
	return
}

// AfterSave also sets the links so that we do not need to
// query the resource directly after creating or updating it.
func (c *Category) AfterSave(tx *gorm.DB) (err error) {
	c.links(tx)
	return
}

func (c *Category) links(tx *gorm.DB) {
	url := tx.Statement.Context.Value(database.ContextURL)

	c.Links.Self = fmt.Sprintf("%s/v1/categories/%s", url, c.ID)
	c.Links.Envelopes = fmt.Sprintf("%s/v1/envelopes?category=%s", url, c.ID)
}
