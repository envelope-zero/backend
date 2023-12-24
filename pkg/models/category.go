package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category represents a category of envelopes.
type Category struct {
	DefaultModel
	CategoryCreate
	Budget   Budget `json:"-"`
	Archived bool   `json:"archived" example:"true" default:"false" gorm:"-"` // Is the Category archived?
}

type CategoryCreate struct {
	Name     string    `json:"name" gorm:"uniqueIndex:category_budget_name" example:"Saving" default:""`                        // Name of the category
	BudgetID uuid.UUID `json:"budgetId" gorm:"uniqueIndex:category_budget_name" example:"52d967d3-33f4-4b04-9ba7-772e5ab9d0ce"` // ID of the budget the category belongs to
	Note     string    `json:"note" example:"All envelopes for long-term saving" default:""`                                    // Notes about the category
	Hidden   bool      `json:"hidden" example:"true" default:"false"`                                                           // Is the category hidden?
}

func (c Category) Self() string {
	return "Category"
}

func (c *Category) BeforeSave(_ *gorm.DB) error {
	c.Name = strings.TrimSpace(c.Name)
	c.Note = strings.TrimSpace(c.Note)

	return nil
}

func (c *Category) AfterFind(_ *gorm.DB) (err error) {
	// Set the Archived field to the value of Hidden
	c.Archived = c.Hidden

	return nil
}

// BeforeUpdate archives all envelopes when the category is archived.
func (c *Category) BeforeUpdate(tx *gorm.DB) (err error) {
	if tx.Statement.Changed("Hidden") && !c.Hidden {
		var envelopes []Envelope
		err = tx.Where(&Envelope{EnvelopeCreate: EnvelopeCreate{
			CategoryID: c.ID,
		}}).
			Find(&envelopes).Error
		if err != nil {
			return
		}

		for _, e := range envelopes {
			e.Hidden = true
			err = tx.Model(&e).Updates(&e).Error
			if err != nil {
				return
			}
		}
	}

	return nil
}

func (c *Category) Envelopes(tx *gorm.DB) ([]Envelope, error) {
	var envelopes []Envelope
	err := tx.Where(&Envelope{EnvelopeCreate: EnvelopeCreate{CategoryID: c.ID}}).Find(&envelopes).Error
	if err != nil {
		return []Envelope{}, err
	}

	return envelopes, nil
}
