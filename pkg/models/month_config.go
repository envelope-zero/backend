package models

import (
	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/google/uuid"
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
	EnvelopeID uuid.UUID   `json:"envelopeId" gorm:"primaryKey" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"` // ID of the envelope
	Month      types.Month `json:"month" gorm:"primaryKey" example:"1969-06-01T00:00:00.000000Z"`               // The month. This is always set to 00:00 UTC on the first of the month.
}

type MonthConfigCreate struct {
	OverspendMode OverspendMode `json:"overspendMode" example:"AFFECT_ENVELOPE" default:"AFFECT_AVAILABLE"`                 // The overspend handling mode to use. Deprecated, will be removed with 4.0.0 release and is not used in API v3 anymore
	Note          string        `json:"note" example:"Added 200€ here because we replaced Tim's expensive vase" default:""` // A note for the month config
}

func (m MonthConfig) Self() string {
	return "Month Config"
}
