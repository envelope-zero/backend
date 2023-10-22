package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Model interface {
	Self() string
}

// DefaultModel is the base model for most models in Envelope Zero.
// As EnvelopeMonth uses the Envelope ID and the Month as primary key,
// the timestamps are managed in the Timestamps struct.
type DefaultModel struct {
	ID uuid.UUID `json:"id" example:"65392deb-5e92-4268-b114-297faad6cdce"` // UUID for the resource
	Timestamps
}

// Timestamps only contains the timestamps that gorm sets automatically to enable other
// primary keys than ID.
type Timestamps struct {
	CreatedAt time.Time       `json:"createdAt" example:"2022-04-02T19:28:44.491514Z"`                                             // Time the resource was created
	UpdatedAt time.Time       `json:"updatedAt" example:"2022-04-17T20:14:01.048145Z"`                                             // Last time the resource was updated
	DeletedAt *gorm.DeletedAt `json:"deletedAt" gorm:"index" example:"2022-04-22T21:01:05.058161Z" swaggertype:"primitive,string"` // Time the resource was marked as deleted
}

// AfterFind updates the timestamps to use UTC as
// timezone, not +0000. Yes, this is different.
//
// We already store them in UTC, but somehow reading
// them from the database returns them as +0000.
func (m *DefaultModel) AfterFind(_ *gorm.DB) (err error) {
	m.CreatedAt = m.CreatedAt.In(time.UTC)
	m.UpdatedAt = m.UpdatedAt.In(time.UTC)

	if m.DeletedAt != nil {
		m.DeletedAt.Time = m.DeletedAt.Time.In(time.UTC)
	}

	return nil
}

// BeforeCreate is set to generate a UUID for the resource.
func (m *DefaultModel) BeforeCreate(_ *gorm.DB) (err error) {
	m.ID = uuid.New()
	return nil
}
