package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
	CreatedAt time.Time `json:"createdAt" example:"2022-04-02T19:28:44.491514Z"` // Time the resource was created
	UpdatedAt time.Time `json:"updatedAt" example:"2022-04-17T20:14:01.048145Z"` // Last time the resource was updated
}

// BeforeCreate is set to generate a UUID for the resource.
func (m *DefaultModel) BeforeCreate(_ *gorm.DB) error {
	m.ID = uuid.New()
	return nil
}
