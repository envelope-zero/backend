package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category represents a category of envelopes.
type Category struct {
	DefaultModel
	CategoryCreate
	Budget Budget `json:"-"` // The budget the category belongs to
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

// BeforeUpdate archives all envelopes when the category is archived.
func (c *Category) BeforeUpdate(tx *gorm.DB) (err error) {
	if tx.Statement.Changed("Hidden") && !c.Hidden {
		var envelopes []Envelope
		err = tx.Model(&Envelope{EnvelopeCreate: EnvelopeCreate{
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
