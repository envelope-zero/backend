package models

import (
	"errors"
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

var ErrCategoryNameNotUnique = errors.New("the category name must be unique for the budget")

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	_ = c.DefaultModel.BeforeCreate(tx)

	toSave := tx.Statement.Dest.(*Category)
	return c.checkIntegrity(tx, *toSave)
}

func (c *Category) BeforeSave(_ *gorm.DB) error {
	c.Name = strings.TrimSpace(c.Name)
	c.Note = strings.TrimSpace(c.Note)

	return nil
}

// BeforeUpdate archives all envelopes when the category is archived.
func (c *Category) BeforeUpdate(tx *gorm.DB) (err error) {
	toSave := tx.Statement.Dest.(Category)
	if tx.Statement.Changed("BudgetID") {
		err := c.checkIntegrity(tx, toSave)
		if err != nil {
			return err
		}
	}

	if tx.Statement.Changed("Archived") && toSave.Archived {
		var envelopes []Envelope
		err = tx.Where(&Envelope{
			CategoryID: c.ID,
		}).
			Find(&envelopes).Error
		if err != nil {
			return
		}

		for _, e := range envelopes {
			err = tx.Model(&e).Select("Archived").Updates(Envelope{Archived: true}).Error
			if err != nil {
				return
			}
		}
	}

	return nil
}

// checkIntegrity verifies references to other resources
func (c *Category) checkIntegrity(tx *gorm.DB, toSave Category) error {
	return tx.First(&Budget{}, toSave.BudgetID).Error
}

func (c *Category) Envelopes(tx *gorm.DB) ([]Envelope, error) {
	var envelopes []Envelope
	err := tx.Where(&Envelope{CategoryID: c.ID}).Find(&envelopes).Error
	if err != nil {
		return []Envelope{}, err
	}

	return envelopes, nil
}
