package models

import (
	"time"

	"gorm.io/gorm"
)

// Model is the base model for all other models in Envelope Zero
type Model struct {
	ID        uint           `json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index"`
}
