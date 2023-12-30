package models

import (
	"strings"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Goal struct {
	DefaultModel
	Name       string `gorm:"uniqueIndex:goal_name_envelope"`
	Note       string
	Envelope   Envelope
	EnvelopeID uuid.UUID       `gorm:"uniqueIndex:goal_name_envelope"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)" example:"750" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The target for the goal
	Month      types.Month
	Archived   bool
}

func (g Goal) Self() string {
	return "Goal"
}

func (g *Goal) BeforeSave(_ *gorm.DB) error {
	g.Name = strings.TrimSpace(g.Name)
	g.Note = strings.TrimSpace(g.Note)

	return nil
}

func (g *Goal) AfterSave(_ *gorm.DB) error {
	if !decimal.Decimal.IsPositive(g.Amount) {
		return ErrGoalAmountNotPositive
	}

	return nil
}
