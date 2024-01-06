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
	EnvelopeID uuid.UUID       `gorm:"primaryKey"` // ID of the envelope
	Month      types.Month     `gorm:"primaryKey"`
	Allocation decimal.Decimal `gorm:"type:DECIMAL(20,8)"`
	Note       string
}

func (m MonthConfig) Self() string {
	return "Month Config"
}

func (m *MonthConfig) BeforeSave(_ *gorm.DB) error {
	m.Note = strings.TrimSpace(m.Note)
	return nil
}
