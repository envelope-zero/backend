package models

import (
	"github.com/envelope-zero/backend/v2/internal/types"
	"github.com/google/uuid"
)

// swagger:enum OverspendMode
type OverspendMode string

const (
	AffectAvailable OverspendMode = "AFFECT_AVAILABLE"
	AffectEnvelope  OverspendMode = "AFFECT_ENVELOPE"
)

type MonthConfig struct {
	Timestamps             // To include the gorm timestamps
	EnvelopeID uuid.UUID   `json:"envelopeId" gorm:"primaryKey" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"`
	Month      types.Month `json:"month" gorm:"primaryKey" example:"1969-06-01T00:00:00.000000Z"` // This is always set to 00:00 UTC on the first of the month.
	MonthConfigCreate
}

type MonthConfigCreate struct {
	OverspendMode OverspendMode `json:"overspendMode" example:"AFFECT_ENVELOPE" default:"AFFECT_AVAILABLE"`
}
