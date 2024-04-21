package v4

import (
	"fmt"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EnvelopeEditable represents all user configurable parameters
type EnvelopeEditable struct {
	Name       string    `json:"name" example:"Groceries" default:""`                                       // Name of the envelope
	CategoryID uuid.UUID `json:"categoryId"  example:"878c831f-af99-4a71-b3ca-80deb7d793c1"`                // ID of the category the envelope belongs to
	Note       string    `json:"note" example:"For stuff bought at supermarkets and drugstores" default:""` // Notes about the envelope
	Archived   bool      `json:"archived" example:"true" default:"false"`                                   // Is the envelope archived?
}

// model transforms the API representation into the model representation
func (e EnvelopeEditable) model() models.Envelope {
	return models.Envelope{
		Name:       e.Name,
		CategoryID: e.CategoryID,
		Note:       e.Note,
		Archived:   e.Archived,
	}
}

type EnvelopeLinks struct {
	Self         string `json:"self" example:"https://example.com/api/v4/envelopes/45b6b5b9-f746-4ae9-b77b-7688b91f8166"`                     // The envelope itself
	Transactions string `json:"transactions" example:"https://example.com/api/v4/transactions?envelope=45b6b5b9-f746-4ae9-b77b-7688b91f8166"` // The envelope's transactions
	Month        string `json:"month" example:"https://example.com/api/v4/envelopes/45b6b5b9-f746-4ae9-b77b-7688b91f8166/YYYY-MM"`            // The MonthConfig for the envelope
}

type Envelope struct {
	models.DefaultModel
	EnvelopeEditable
	Links EnvelopeLinks `json:"links"` // Links to related resources
}

func newEnvelope(c *gin.Context, model models.Envelope) Envelope {
	url := c.GetString(string(models.DBContextURL))

	return Envelope{
		DefaultModel: model.DefaultModel,
		EnvelopeEditable: EnvelopeEditable{
			Name:       model.Name,
			CategoryID: model.CategoryID,
			Note:       model.Note,
			Archived:   model.Archived,
		},
		Links: EnvelopeLinks{
			Self:         fmt.Sprintf("%s/v4/envelopes/%s", url, model.ID),
			Transactions: fmt.Sprintf("%s/v4/transactions?envelope=%s", url, model.ID),
			Month:        fmt.Sprintf("%s/v4/envelopes/%s/YYYY-MM", url, model.ID),
		},
	}
}

type EnvelopeListResponse struct {
	Data       []Envelope  `json:"data"`                                                          // List of Envelopes
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type EnvelopeCreateResponse struct {
	Data  []EnvelopeResponse `json:"data"`                                                          // Data for the Envelope
	Error *string            `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

// appendError appends an EnvelopeResponse with the error and returns the updated HTTP status
func (e *EnvelopeCreateResponse) appendError(err error, currentStatus int) int {
	s := err.Error()
	e.Data = append(e.Data, EnvelopeResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	newStatus := status(err)
	if newStatus > currentStatus {
		return newStatus
	}

	return currentStatus
}

type EnvelopeResponse struct {
	Data  *Envelope `json:"data"`                                                          // Data for the Envelope
	Error *string   `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type EnvelopeQueryFilter struct {
	BudgetID   string `form:"budget" filterField:"false"` // By budget ID
	CategoryID string `form:"category"`                   // By the ID of the category
	Name       string `form:"name" filterField:"false"`   // By name
	Note       string `form:"note" filterField:"false"`   // By the note
	Archived   bool   `form:"archived"`                   // Is the envelope archived?
	Search     string `form:"search" filterField:"false"` // By string in name or note
	Offset     uint   `form:"offset" filterField:"false"` // The offset of the first Envelope returned. Defaults to 0.
	Limit      int    `form:"limit" filterField:"false"`  // Maximum number of Envelopes to return. Defaults to 50.
}

func (f EnvelopeQueryFilter) model() (models.Envelope, error) {
	categoryID, err := httputil.UUIDFromString(f.CategoryID)
	if err != nil {
		return models.Envelope{}, err
	}

	return models.Envelope{
		CategoryID: categoryID,
		Archived:   f.Archived,
	}, nil
}
