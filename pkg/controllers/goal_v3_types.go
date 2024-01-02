package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/database"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type GoalV3Editable struct {
	Name       string          `json:"name" example:"New TV" default:""`                                                                              // Name of the goal
	Note       string          `json:"note" example:"We want to replace the old CRT TV soon-ish" default:""`                                          // Note about the goal
	EnvelopeID uuid.UUID       `json:"envelopeId" example:"f81566d9-af4d-4f13-9830-c62c4b5e4c7e"`                                                     // The ID of the envelope this goal is for
	Amount     decimal.Decimal `json:"amount" example:"750" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001" default:"0"` // How much money should be saved for this goal?
	Month      types.Month     `json:"month" example:"2024-07-01T00:00:00.000000Z"`                                                                   // The month the goal should be reached
	Archived   bool            `json:"archived" example:"true" default:"false"`                                                                       // If this goal is still in use or not
}

// model returns the database resource for the API representation of the editable fields
func (editable GoalV3Editable) model() models.Goal {
	return models.Goal{
		Name:       editable.Name,
		Note:       editable.Note,
		EnvelopeID: editable.EnvelopeID,
		Amount:     editable.Amount,
		Month:      editable.Month,
		Archived:   editable.Archived,
	}
}

type GoalV3Links struct {
	Self     string `json:"self" example:"https://example.com/api/v3/goals/438cc6c0-9baf-49fd-a75a-d76bd5cab19c"`         // The Goal itself
	Envelope string `json:"envelope" example:"https://example.com/api/v3/envelopes/c1a96ae4-80e3-4827-8ed0-c7656f224fee"` // The Envelope this goal references
}

type GoalV3 struct {
	models.DefaultModel
	GoalV3Editable
	Links GoalV3Links `json:"links"`
}

// newGoalV3 returns the API v3 representation of the resource
func newGoalV3(c *gin.Context, model models.Goal) GoalV3 {
	url := c.GetString(string(database.ContextURL))

	return GoalV3{
		DefaultModel: model.DefaultModel,
		GoalV3Editable: GoalV3Editable{
			Name:       model.Name,
			Note:       model.Note,
			EnvelopeID: model.EnvelopeID,
			Amount:     model.Amount,
			Month:      model.Month,
			Archived:   model.Archived,
		},
		Links: GoalV3Links{
			Self:     fmt.Sprintf("%s/v3/goals/%s", url, model.ID),
			Envelope: fmt.Sprintf("%s/v3/envelopes/%s", url, model.EnvelopeID),
		},
	}
}

type GoalListResponseV3 struct {
	Data       []GoalV3    `json:"data"`                                                          // List of resources
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type GoalCreateResponseV3 struct {
	Error *string          `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []GoalResponseV3 `json:"data"`                                                          // List of created resources
}

func (t *GoalCreateResponseV3) appendError(err httperrors.Error, status int) int {
	s := err.Error()
	t.Data = append(t.Data, GoalResponseV3{Error: &s})

	// The final status code is the highest HTTP status code number
	if err.Status > status {
		status = err.Status
	}

	return status
}

type GoalResponseV3 struct {
	Error *string `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  *GoalV3 `json:"data"`                                                          // The resource
}

type GoalQueryFilterV3 struct {
	Name              string          `form:"name" filterField:"false"`              // By name
	Note              string          `form:"note" filterField:"false"`              // By the note
	Search            string          `form:"search" filterField:"false"`            // By string in name or note
	Archived          bool            `form:"archived"`                              // Is the goal archived?
	EnvelopeID        string          `form:"envelope"`                              // ID of the envelope
	Month             string          `form:"month"`                                 // Exact month
	FromMonth         string          `form:"fromMonth" filterField:"false"`         // From this month
	UntilMonth        string          `form:"untilMonth" filterField:"false"`        // Until this month
	Amount            decimal.Decimal `form:"amount"`                                // Exact amount
	AmountLessOrEqual decimal.Decimal `form:"amountLessOrEqual" filterField:"false"` // Amount less than or equal to this
	AmountMoreOrEqual decimal.Decimal `form:"amountMoreOrEqual" filterField:"false"` // Amount more than or equal to this
	Offset            uint            `form:"offset" filterField:"false"`            // The offset of the first goal returned. Defaults to 0.
	Limit             int             `form:"limit" filterField:"false"`             // Maximum number of goals to return. Defaults to 50.
}

func (f GoalQueryFilterV3) model() (models.Goal, httperrors.Error) {
	envelopeID, err := httputil.UUIDFromString(f.EnvelopeID)
	if !err.Nil() {
		return models.Goal{}, err
	}

	var month types.Month
	if f.Month != "" {
		m, e := types.ParseMonth(f.Month)
		if e != nil {
			return models.Goal{}, httperrors.Error{
				Err:    e,
				Status: http.StatusBadRequest,
			}
		}

		month = m
	}

	// This does not set the string fields since they are
	// handled in the controller function
	return GoalV3Editable{
		EnvelopeID: envelopeID,
		Amount:     f.Amount,
		Month:      month,
		Archived:   f.Archived,
	}.model(), httperrors.Error{}
}
