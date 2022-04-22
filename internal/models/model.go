package models

import (
	"time"

	"gorm.io/gorm"
)

// Model is the base model for all other models in Envelope Zero.
type Model struct {
	ID        uint64          `json:"id" example:"42" format:"uint64"`
	CreatedAt time.Time       `json:"createdAt" example:"2022-04-02T19:28:44.491514Z"`
	UpdatedAt time.Time       `json:"updatedAt" example:"2022-04-17T20:14:01.048145Z"`
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
