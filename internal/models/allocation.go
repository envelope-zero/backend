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
	Month      uint8           `json:"month" gorm:"uniqueIndex:year_month;check:month_valid,month >= 1 AND month <= 12"`
	Year       uint            `json:"year" gorm:"uniqueIndex:year_month"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)"`
	EnvelopeID uuid.UUID       `json:"envelopeId,omitempty"`
}
