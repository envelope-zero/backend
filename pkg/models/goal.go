package models

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Goal struct {
	DefaultModel
	Name       string `gorm:"uniqueIndex:goal_name_envelope"`
	Note       string
	Envelope   Envelope        `json:"-"`
	EnvelopeID uuid.UUID       `gorm:"uniqueIndex:goal_name_envelope"`
	Amount     decimal.Decimal `gorm:"type:DECIMAL(20,8)"` // The target for the goal
	Month      types.Month
	Archived   bool
}

var ErrGoalAmountNotPositive = errors.New("goal amounts must be larger than zero")

func (g *Goal) BeforeCreate(tx *gorm.DB) error {
	_ = g.DefaultModel.BeforeCreate(tx)

	toSave := tx.Statement.Dest.(*Goal)
	return g.checkIntegrity(tx, *toSave)
}

func (g *Goal) BeforeUpdate(tx *gorm.DB) (err error) {
	toSave := tx.Statement.Dest.(Goal)

	if tx.Statement.Changed("EnvelopeID") {
		err := g.checkIntegrity(tx, toSave)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Goal) checkIntegrity(tx *gorm.DB, toSave Goal) error {
	return tx.First(&Envelope{}, toSave.EnvelopeID).Error
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

// Returns all goals on this instance for export
func (Goal) Export() (json.RawMessage, error) {
	var goals []Goal
	err := DB.Unscoped().Where(&Goal{}).Find(&goals).Error
	if err != nil {
		return nil, err
	}

	j, err := json.Marshal(&goals)
	if err != nil {
		return json.RawMessage{}, err
	}
	return json.RawMessage(j), nil
}
