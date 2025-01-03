package v4

import (
	"fmt"

	"github.com/envelope-zero/backend/v5/internal/models"
	"github.com/envelope-zero/backend/v5/internal/types"
	ez_uuid "github.com/envelope-zero/backend/v5/internal/uuid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type GoalEditable struct {
	Name       string          `json:"name" example:"New TV" default:""`                                                                              // Name of the goal
	Note       string          `json:"note" example:"We want to replace the old CRT TV soon-ish" default:""`                                          // Note about the goal
	EnvelopeID uuid.UUID       `json:"envelopeId" example:"f81566d9-af4d-4f13-9830-c62c4b5e4c7e"`                                                     // The ID of the envelope this goal is for
	Amount     decimal.Decimal `json:"amount" example:"750" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001" default:"0"` // How much money should be saved for this goal?
	Month      types.Month     `json:"month" example:"2024-07-01T00:00:00.000000Z"`                                                                   // The month the goal should be reached
	Archived   bool            `json:"archived" example:"true" default:"false"`                                                                       // If this goal is still in use or not
}

// model returns the database resource for the API representation of the editable fields
func (editable GoalEditable) model() models.Goal {
	return models.Goal{
		Name:       editable.Name,
		Note:       editable.Note,
		EnvelopeID: editable.EnvelopeID,
		Amount:     editable.Amount,
		Month:      editable.Month,
		Archived:   editable.Archived,
	}
}

type GoalLinks struct {
	Self     string `json:"self" example:"https://example.com/api/v4/goals/438cc6c0-9baf-49fd-a75a-d76bd5cab19c"`         // The Goal itself
	Envelope string `json:"envelope" example:"https://example.com/api/v4/envelopes/c1a96ae4-80e3-4827-8ed0-c7656f224fee"` // The Envelope this goal references
}

type Goal struct {
	models.DefaultModel
	GoalEditable
	Links GoalLinks `json:"links"`
}

// newGoal returns the API v4 representation of the resource
func newGoal(c *gin.Context, model models.Goal) Goal {
	url := c.GetString(string(models.DBContextURL))

	return Goal{
		DefaultModel: model.DefaultModel,
		GoalEditable: GoalEditable{
			Name:       model.Name,
			Note:       model.Note,
			EnvelopeID: model.EnvelopeID,
			Amount:     model.Amount,
			Month:      model.Month,
			Archived:   model.Archived,
		},
		Links: GoalLinks{
			Self:     fmt.Sprintf("%s/v4/goals/%s", url, model.ID),
			Envelope: fmt.Sprintf("%s/v4/envelopes/%s", url, model.EnvelopeID),
		},
	}
}

type GoalListResponse struct {
	Data       []Goal      `json:"data"`                                                          // List of resources
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type GoalCreateResponse struct {
	Error *string        `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []GoalResponse `json:"data"`                                                          // List of created resources
}

func (t *GoalCreateResponse) appendError(err error, currentStatus int) int {
	s := err.Error()
	t.Data = append(t.Data, GoalResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	newStatus := status(err)
	if newStatus > currentStatus {
		return newStatus
	}

	return currentStatus
}

type GoalResponse struct {
	Error *string `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  *Goal   `json:"data"`                                                          // The resource
}

type GoalQueryFilter struct {
	BudgetID          ez_uuid.UUID    `form:"budget" filterField:"false"`            // By budget ID
	CategoryID        ez_uuid.UUID    `form:"category" filterField:"false"`          // By category ID
	Name              string          `form:"name" filterField:"false"`              // By name
	Note              string          `form:"note" filterField:"false"`              // By the note
	Search            string          `form:"search" filterField:"false"`            // By string in name or note
	Archived          bool            `form:"archived"`                              // Is the goal archived?
	EnvelopeID        ez_uuid.UUID    `form:"envelope"`                              // ID of the envelope
	Month             string          `form:"month"`                                 // Exact month
	FromMonth         string          `form:"fromMonth" filterField:"false"`         // From this month
	UntilMonth        string          `form:"untilMonth" filterField:"false"`        // Until this month
	Amount            decimal.Decimal `form:"amount"`                                // Exact amount
	AmountLessOrEqual decimal.Decimal `form:"amountLessOrEqual" filterField:"false"` // Amount less than or equal to this
	AmountMoreOrEqual decimal.Decimal `form:"amountMoreOrEqual" filterField:"false"` // Amount more than or equal to this
	Offset            uint            `form:"offset" filterField:"false"`            // The offset of the first goal returned. Defaults to 0.
	Limit             int             `form:"limit" filterField:"false"`             // Maximum number of goals to return. Defaults to 50.
}

func (f GoalQueryFilter) model() (models.Goal, error) {
	var month types.Month
	if f.Month != "" {
		m, err := types.ParseMonth(f.Month)
		if err != nil {
			return models.Goal{}, err
		}

		month = m
	}

	// This does not set the string fields since they are
	// handled in the controller function
	return GoalEditable{
		EnvelopeID: f.EnvelopeID.UUID,
		Amount:     f.Amount,
		Month:      month,
		Archived:   f.Archived,
	}.model(), nil
}
