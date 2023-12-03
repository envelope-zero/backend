package models

import (
	"errors"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// swagger:enum OverspendMode
type OverspendMode string

const (
	AffectAvailable OverspendMode = "AFFECT_AVAILABLE"
	AffectEnvelope  OverspendMode = "AFFECT_ENVELOPE"
)

type MonthConfig struct {
	Timestamps
	MonthConfigCreate
	EnvelopeID uuid.UUID       `json:"envelopeId" gorm:"primaryKey" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"`                                      // ID of the envelope
	Month      types.Month     `json:"month" gorm:"primaryKey" example:"1969-06-01T00:00:00.000000Z"`                                                    // The month. This is always set to 00:00 UTC on the first of the month.
	Allocation decimal.Decimal `json:"allocation" gorm:"-" example:"22.01" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
}

type MonthConfigCreate struct {
	OverspendMode OverspendMode `json:"overspendMode" example:"AFFECT_ENVELOPE" default:"AFFECT_AVAILABLE"`                 // The overspend handling mode to use. Deprecated, will be removed with 4.0.0 release and is not used in API v3 anymore
	Note          string        `json:"note" example:"Added 200€ here because we replaced Tim's expensive vase" default:""` // A note for the month config
}

func (m MonthConfig) Self() string {
	return "Month Config"
}

func (m *MonthConfig) AfterFind(tx *gorm.DB) error {
	// Check if there is an allocation for this MonthConfig. If yes, set the value.
	// This transparently makes use of the Allocation model
	var a Allocation
	err := tx.First(&a, Allocation{
		AllocationCreate: AllocationCreate{
			Month:      m.Month,
			EnvelopeID: m.EnvelopeID,
		},
	}).Error

	// If there is a database error, return it
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Set the amount if there is an allocation. If not,
	// the amount is 0, which is the zero value of decimal.Decimal
	if err == nil {
		m.Allocation = a.Amount
	}

	return nil
}
