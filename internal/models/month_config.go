package models

import (
	"encoding/json"
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

// Returns all match rules on this instance for export
func (MonthConfig) Export() (json.RawMessage, error) {
	var monthConfigs []MonthConfig
	err := DB.Unscoped().Where(&MonthConfig{}).Find(&monthConfigs).Error
	if err != nil {
		return nil, err
	}

	j, err := json.Marshal(&monthConfigs)
	if err != nil {
		return json.RawMessage{}, err
	}
	return json.RawMessage(j), nil
}
