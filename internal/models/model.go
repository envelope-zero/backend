package models

import (
	"time"

	"gorm.io/gorm"
)

// Model is the base model for all other models in Envelope Zero.
type Model struct {
	ID        uint64          `json:"id"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	DeletedAt *gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// AfterFind updates the timestamps to use UTC as
// timezone, not +0000. Yes, this is different.
//
// We already store them in UTC, but somehow reading
// them from the database returns them as +0000.
func (m *Model) AfterFind(tx *gorm.DB) (err error) {
	m.CreatedAt = m.CreatedAt.In(time.UTC)
	m.UpdatedAt = m.UpdatedAt.In(time.UTC)

	if m.DeletedAt != nil {
		m.DeletedAt.Time = m.DeletedAt.Time.In(time.UTC)
	}

	return nil
}
