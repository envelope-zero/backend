package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category represents a category of envelopes.
type Category struct {
	DefaultModel
	Budget   Budget
	BudgetID uuid.UUID `gorm:"uniqueIndex:category_budget_name"`
	Name     string    `gorm:"uniqueIndex:category_budget_name"`
	Note     string
	Archived bool
}

func (c Category) Self() string {
	return "Category"
}

func (c *Category) BeforeSave(_ *gorm.DB) error {
	c.Name = strings.TrimSpace(c.Name)
	c.Note = strings.TrimSpace(c.Note)

	return nil
}

// BeforeUpdate archives all envelopes when the category is archived.
func (c *Category) BeforeUpdate(tx *gorm.DB) (err error) {
	if tx.Statement.Changed("Archived") && !c.Archived {
		var envelopes []Envelope
		err = tx.Where(&Envelope{
			CategoryID: c.ID,
		}).
			Find(&envelopes).Error
		if err != nil {
			return
		}

		for _, e := range envelopes {
			e.Archived = true
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
	err := tx.Where(&Envelope{CategoryID: c.ID}).Find(&envelopes).Error
	if err != nil {
		return []Envelope{}, err
	}

	return envelopes, nil
}
