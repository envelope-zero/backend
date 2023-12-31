package models

import (
	"strings"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type MonthConfig struct {
	Timestamps
	MonthConfigCreate
	EnvelopeID uuid.UUID   `json:"envelopeId" gorm:"primaryKey" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"` // ID of the envelope
	Month      types.Month `json:"month" gorm:"primaryKey" example:"1969-06-01T00:00:00.000000Z"`               // The month. This is always set to 00:00 UTC on the first of the month.
}

type MonthConfigCreate struct {
	Allocation decimal.Decimal `json:"allocation" gorm:"type:DECIMAL(20,8)" example:"22.01" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
	Note       string          `json:"note" example:"Added 200€ here because we replaced Tim's expensive vase" default:""`                                                // A note for the month config
}

func (m MonthConfig) Self() string {
	return "Month Config"
}

func (m *MonthConfig) BeforeSave(_ *gorm.DB) error {
	m.Note = strings.TrimSpace(m.Note)
	return nil
}
