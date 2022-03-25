package models

import (
	"github.com/shopspring/decimal"
)

// Allocation represents the allocation of money to an Envelope for a specific month.
type Allocation struct {
	Model
	Month      uint8           `json:"month" gorm:"uniqueIndex:year_month;check:month >= 1 AND month <= 12"`
	Year       uint            `json:"year" gorm:"uniqueIndex:year_month"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)"`
	EnvelopeID int             `json:"envelopeId,omitempty"`
	Envelope   Envelope        `json:"-"`
}
