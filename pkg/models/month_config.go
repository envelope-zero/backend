package models

import (
	"errors"
	"strings"

	"github.com/envelope-zero/backend/v5/internal/types"
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

var ErrMonthConfigMonthNotUnique = errors.New("you can not create multiple month configs for the same envelope and month")

func (m *MonthConfig) BeforeSave(_ *gorm.DB) error {
	m.Note = strings.TrimSpace(m.Note)
	return nil
}
