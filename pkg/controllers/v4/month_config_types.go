package v4

import (
	"fmt"

	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// swagger:enum AllocationMode
type AllocationMode string

const (
	AllocateLastMonthBudget AllocationMode = "ALLOCATE_LAST_MONTH_BUDGET"
	AllocateLastMonthSpend  AllocationMode = "ALLOCATE_LAST_MONTH_SPEND"
)

type BudgetAllocationMode struct {
	Mode AllocationMode `json:"mode" example:"ALLOCATE_LAST_MONTH_SPEND"` // Mode to allocate budget with
}

type MonthConfigEditable struct {
	EnvelopeID uuid.UUID       `json:"envelopeId" gorm:"primaryKey" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"`                                      // ID of the envelope
	Month      types.Month     `json:"month" gorm:"primaryKey" example:"1969-06-01T00:00:00.000000Z"`                                                    // The month. This is always set to 00:00 UTC on the first of the month.
	Allocation decimal.Decimal `json:"allocation" gorm:"-" example:"22.01" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
	Note       string          `json:"note" example:"Added 200€ here because we replaced Tim's expensive vase" default:""`                               // A note for the month config
}

func (editable MonthConfigEditable) model() models.MonthConfig {
	return models.MonthConfig{
		EnvelopeID: editable.EnvelopeID,
		Month:      editable.Month,
		Allocation: editable.Allocation,
		Note:       editable.Note,
	}
}

type MonthConfigLinks struct {
	Self     string `json:"self" example:"https://example.com/api/v4/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b/2017-10"` // The Month Config itself
	Envelope string `json:"envelope" example:"https://example.com/api/v4/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b"`     // The Envelope this config belongs to
}

type MonthConfig struct {
	MonthConfigEditable
	EnvelopeID uuid.UUID        // We do not use the default model here, we use envelope ID and month
	Month      types.Month      // We do not use the default model here, we use envelope ID and month
	Links      MonthConfigLinks `json:"links"`
}

func newMonthConfig(c *gin.Context, model models.MonthConfig) MonthConfig {
	url := c.GetString(string(models.DBContextURL))

	return MonthConfig{
		EnvelopeID: model.EnvelopeID,
		Month:      model.Month,
		MonthConfigEditable: MonthConfigEditable{
			EnvelopeID: model.EnvelopeID,
			Month:      model.Month,
			Allocation: model.Allocation,
			Note:       model.Note,
		},
		Links: MonthConfigLinks{
			Self:     fmt.Sprintf("%s/v4/envelopes/%s/%s", url, model.EnvelopeID, model.Month),
			Envelope: fmt.Sprintf("%s/v4/envelopes/%s", url, model.EnvelopeID),
		},
	}
}

// getMonthConfigModel returns the month config for a specific envelope and month
//
// It is the month config equivalent for getModelByID
func getMonthConfigModel(id uuid.UUID, month types.Month) (models.MonthConfig, error) {
	var m models.MonthConfig

	err := models.DB.First(&m, &models.MonthConfig{
		EnvelopeID: id,
		Month:      month,
	}).Error
	if err != nil {
		return models.MonthConfig{}, err
	}

	return m, nil
}

type MonthConfigResponse struct {
	Data  *MonthConfig `json:"data"`                                                          // Config for the month
	Error *string      `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type MonthConfigListResponse struct {
	Data       []MonthConfig `json:"data"`                                                          // List of Month Configs
	Error      *string       `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination   `json:"pagination"`                                                    // Pagination information
}
