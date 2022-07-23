package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Allocation represents the allocation of money to an Envelope for a specific month.
type Allocation struct {
	Model
	AllocationCreate
	Envelope Envelope `json:"-"`
}

type AllocationCreate struct {
	Month      uint8           `json:"month" gorm:"uniqueIndex:year_month;check:month_valid,month >= 1 AND month <= 12" minimum:"1" maximum:"12" example:"6"`
	Year       uint            `json:"year" gorm:"uniqueIndex:year_month" example:"2022"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)" example:"22.01" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
	EnvelopeID uuid.UUID       `json:"envelopeId" example:"a0909e84-e8f9-4cb6-82a5-025dff105ff2"`
}
