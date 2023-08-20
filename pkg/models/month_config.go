package models

import (
	"fmt"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// swagger:enum OverspendMode
type OverspendMode string

const (
	AffectAvailable OverspendMode = "AFFECT_AVAILABLE"
	AffectEnvelope  OverspendMode = "AFFECT_ENVELOPE"
)

type MonthConfig struct {
	Timestamps
	MonthConfigCreate
	EnvelopeID uuid.UUID   `json:"envelopeId" gorm:"primaryKey" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"` // ID of the envelope
	Month      types.Month `json:"month" gorm:"primaryKey" example:"1969-06-01T00:00:00.000000Z"`               // The month. This is always set to 00:00 UTC on the first of the month.
	Links      struct {
		Self     string `json:"self" example:"https://example.com/api/v1/month-configs/61027ebb-ab75-4a49-9e23-a104ddd9ba6b/2017-10"` // The month config itself
		Envelope string `json:"envelope" example:"https://example.com/api/v1/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b"`         // The envelope this config belongs to
	} `json:"links" gorm:"-"`
}

type MonthConfigCreate struct {
	OverspendMode OverspendMode `json:"overspendMode" example:"AFFECT_ENVELOPE" default:"AFFECT_AVAILABLE"`                 // The overspend handling mode to use
	Note          string        `json:"note" example:"Added 200€ here because we replaced Tim's expensive vase" default:""` // A note for the month config
}

func (m MonthConfig) Self() string {
	return "Month Config"
}

// AfterSave also sets the links so that we do not need to
// query the resource directly after creating or updating it.
func (m *MonthConfig) AfterSave(tx *gorm.DB) (err error) {
	m.links(tx)
	return
}

func (m *MonthConfig) AfterFind(tx *gorm.DB) (err error) {
	m.links(tx)
	return
}

func (m *MonthConfig) links(tx *gorm.DB) {
	url := tx.Statement.Context.Value(database.ContextURL)

	m.Links.Self = fmt.Sprintf("%s/v1/month-configs/%s/%s", url, m.EnvelopeID, m.Month)
	m.Links.Envelope = fmt.Sprintf("%s/v1/envelopes/%s", url, m.EnvelopeID)
}
