package models

import (
	"github.com/envelope-zero/backend/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Allocation represents the allocation of money to an Envelope for a specific month.
type Allocation struct {
	DefaultModel
	AllocationCreate
	Envelope Envelope `json:"-"`
}

type AllocationCreate struct {
	Month      types.Month     `json:"month" gorm:"uniqueIndex:allocation_month_envelope" example:"2021-12-01T00:00:00.000000Z"`                                      // Only year and month of this timestamp are used, everything else is ignored. This will always be set to 00:00 UTC on the first of the specified month
	Amount     decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)" example:"22.01" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
	EnvelopeID uuid.UUID       `json:"envelopeId" gorm:"uniqueIndex:allocation_month_envelope" example:"a0909e84-e8f9-4cb6-82a5-025dff105ff2"`
}
