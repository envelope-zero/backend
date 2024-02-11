package v4

import (
	"fmt"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchRuleEditable struct {
	AccountID uuid.UUID `json:"accountId" example:"f9e873c2-fb96-4367-bfb6-7ecd9bf4a6b5"` // The account to map matching transactions to
	Priority  uint      `json:"priority" example:"3"`                                     // The priority of the match rule
	Match     string    `json:"match" example:"Bank*"`                                    // The matching applied to the opposite account. This is a glob pattern. Multiple globs are allowed. Globbing is case sensitive.
}

func (editable MatchRuleEditable) model() models.MatchRule {
	return models.MatchRule{
		AccountID: editable.AccountID,
		Priority:  editable.Priority,
		Match:     editable.Match,
	}
}

type MatchRuleListResponse struct {
	Data       []MatchRule `json:"data"`                                                          // List of Match Rules
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type MatchRuleCreateResponse struct {
	Error *string             `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []MatchRuleResponse `json:"data"`                                                          // List of created Match Rules
}

func (m *MatchRuleCreateResponse) appendError(err error, currentStatus int) int {
	s := err.Error()
	m.Data = append(m.Data, MatchRuleResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	newStatus := status(err)
	if newStatus > currentStatus {
		return newStatus
	}

	return currentStatus
}

type MatchRuleResponse struct {
	Error *string    `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this Match Rule
	Data  *MatchRule `json:"data"`                                                          // The Match Rule data, if creation was successful
}

type MatchRuleLinks struct {
	Self string `json:"self" example:"https://example.com/api/v4/match-rules/95685c82-53c6-455d-b235-f49960b73b21"` // The match rule itself
}

// MatchRule is the API representation of a Match Rule.
type MatchRule struct {
	models.DefaultModel
	MatchRuleEditable
	Links MatchRuleLinks `json:"links"`
}

func newMatchRule(c *gin.Context, model models.MatchRule) MatchRule {
	url := c.GetString(string(models.DBContextURL))

	return MatchRule{
		DefaultModel: model.DefaultModel,
		MatchRuleEditable: MatchRuleEditable{
			AccountID: model.AccountID,
			Priority:  model.Priority,
			Match:     model.Match,
		},
		Links: MatchRuleLinks{
			Self: fmt.Sprintf("%s/v4/match-rules/%s", url, model.ID),
		},
	}
}

// MatchRuleQueryFilter contains the fields that Match Rules can be filtered with.
type MatchRuleQueryFilter struct {
	Priority  uint   `form:"priority"`                   // By priority
	Match     string `form:"match" filterField:"false"`  // By match
	AccountID string `form:"account"`                    // By ID of the Account they map to
	Offset    uint   `form:"offset" filterField:"false"` // The offset of the first Match Rule returned. Defaults to 0.
	Limit     int    `form:"limit" filterField:"false"`  // Maximum number of Match Rules to return. Defaults to 50.
}

// Parse returns a models.MatchRuleCreate struct that represents the MatchRuleQueryFilter.
func (f MatchRuleQueryFilter) model() (models.MatchRule, error) {
	envelopeID, err := httputil.UUIDFromString(f.AccountID)
	if err != nil {
		return models.MatchRule{}, err
	}

	return models.MatchRule{
		Priority:  f.Priority,
		AccountID: envelopeID,
	}, nil
}
