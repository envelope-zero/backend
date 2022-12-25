package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/internal/types"
	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

type EnvelopeListResponse struct {
	Data []Envelope `json:"data"`
}

type EnvelopeResponse struct {
	Data Envelope `json:"data"`
}

type Envelope struct {
	models.Envelope
	Links EnvelopeLinks `json:"links"`
}

type EnvelopeMonthResponse struct {
	Data models.EnvelopeMonth `json:"data"`
}

type EnvelopeLinks struct {
	Self         string `json:"self" example:"https://example.com/api/v1/envelopes/45b6b5b9-f746-4ae9-b77b-7688b91f8166"`
	Allocations  string `json:"allocations" example:"https://example.com/api/v1/allocations?envelope=45b6b5b9-f746-4ae9-b77b-7688b91f8166"`
	Month        string `json:"month" example:"https://example.com/api/v1/envelopes/45b6b5b9-f746-4ae9-b77b-7688b91f8166/YYYY-MM"` // This will always end in 'YYYY-MM' for clients to use replace with actual numbers.
	Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions?envelope=45b6b5b9-f746-4ae9-b77b-7688b91f8166"`
}

type EnvelopeQueryFilter struct {
	Name       string `form:"name" filterField:"false"`
	CategoryID string `form:"category"`
	Note       string `form:"note" filterField:"false"`
}

func (e EnvelopeQueryFilter) ToCreate(c *gin.Context) (models.EnvelopeCreate, bool) {
	categoryID, ok := httputil.UUIDFromString(c, e.CategoryID)
	if !ok {
		return models.EnvelopeCreate{}, false
	}

	return models.EnvelopeCreate{
		CategoryID: categoryID,
	}, true
}

// RegisterEnvelopeRoutes registers the routes for envelopes with
// the RouterGroup that is passed.
func (co Controller) RegisterEnvelopeRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsEnvelopeList)
		r.GET("", co.GetEnvelopes)
		r.POST("", co.CreateEnvelope)
	}

	// Envelope with ID
	{
		r.OPTIONS("/:envelopeId", co.OptionsEnvelopeDetail)
		r.GET("/:envelopeId", co.GetEnvelope)
		r.GET("/:envelopeId/:month", co.GetEnvelopeMonth)
		r.PATCH("/:envelopeId", co.UpdateEnvelope)
		r.DELETE("/:envelopeId", co.DeleteEnvelope)
	}
}

// OptionsEnvelopeList returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Envelopes
//	@Success		204
//	@Router			/v1/envelopes [options]
func (co Controller) OptionsEnvelopeList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsEnvelopeDetail returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Envelopes
//	@Success		204
//	@Param			envelopeId	path	string	true	"ID formatted as string"
//	@Router			/v1/envelopes/{envelopeId} [options]
func (co Controller) OptionsEnvelopeDetail(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := co.getEnvelopeObject(c, p)
	if !ok {
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// CreateEnvelope creates a new envelope
//
//	@Summary		Create envelope
//	@Description	Creates a new envelope
//	@Tags			Envelopes
//	@Produce		json
//	@Success		201	{object}	EnvelopeResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelope	body		models.EnvelopeCreate	true	"Envelope"
//	@Router			/v1/envelopes [post]
func (co Controller) CreateEnvelope(c *gin.Context) {
	var envelope models.Envelope

	err := httputil.BindData(c, &envelope)
	if err != nil {
		return
	}

	_, ok := co.getCategoryResource(c, envelope.CategoryID)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Create(&envelope)) {
		return
	}

	envelopeObject, _ := co.getEnvelopeObject(c, envelope.ID)
	c.JSON(http.StatusCreated, EnvelopeResponse{Data: envelopeObject})
}

// GetEnvelopes returns a list of envelopes filtered by the query parameters
//
//	@Summary		Get envelopes
//	@Description	Returns a list of envelopes
//	@Tags			Envelopes
//	@Produce		json
//	@Success		200	{object}	EnvelopeListResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/envelopes [get]
//	@Param			name		query	string	false	"Filter by name"
//	@Param			note		query	string	false	"Filter by note"
//	@Param			category	query	string	false	"Filter by category ID"
func (co Controller) GetEnvelopes(c *gin.Context) {
	var filter EnvelopeQueryFilter

	// The filters contain only strings, so this will always succeed
	_ = c.Bind(&filter)

	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, ok := filter.ToCreate(c)
	if !ok {
		return
	}

	query := co.DB.Where(&models.Envelope{
		EnvelopeCreate: create,
	}, queryFields...)

	if filter.Name != "" {
		query = query.Where("name LIKE ?", fmt.Sprintf("%%%s%%", filter.Name))
	} else if slices.Contains(setFields, "Name") {
		query = query.Where("name = ''")
	}

	if filter.Note != "" {
		query = query.Where("note LIKE ?", fmt.Sprintf("%%%s%%", filter.Note))
	} else if slices.Contains(setFields, "Note") {
		query = query.Where("note = ''")
	}

	var envelopes []models.Envelope
	if !queryWithRetry(c, query.Find(&envelopes)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	envelopeObjects := make([]Envelope, 0)

	for _, envelope := range envelopes {
		o, _ := co.getEnvelopeObject(c, envelope.ID)
		envelopeObjects = append(envelopeObjects, o)
	}

	c.JSON(http.StatusOK, EnvelopeListResponse{Data: envelopeObjects})
}

// GetEnvelope returns data about a specific envelope
//
//	@Summary		Get envelope
//	@Description	Returns a specific envelope
//	@Tags			Envelopes
//	@Produce		json
//	@Success		200	{object}	EnvelopeResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string	true	"ID formatted as string"
//	@Router			/v1/envelopes/{envelopeId} [get]
func (co Controller) GetEnvelope(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	envelopeObject, ok := co.getEnvelopeObject(c, p)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, EnvelopeResponse{Data: envelopeObject})
}

// GetEnvelopeMonth returns month data for a specific envelope
//
//	@Summary		Get Envelope month data
//	@Description	Returns data about an envelope for a for a specific month. **Use GET /month endpoint with month and budgetId query parameters instead.**
//	@Tags			Envelopes
//	@Produce		json
//	@Success		200	{object}	EnvelopeMonthResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string	true	"ID formatted as string"
//	@Param			month		path		string	true	"The month in YYYY-MM format"
//	@Router			/v1/envelopes/{envelopeId}/{month} [get]
//	@Deprecated		true
func (co Controller) GetEnvelopeMonth(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	envelope, ok := co.getEnvelopeResource(c, p)
	if !ok {
		httperrors.Handler(c, err)
		return
	}

	if month.Month.IsZero() {
		httperrors.New(c, http.StatusBadRequest, "You cannot request data for no month")
		return
	}

	envelopeMonth, _, err := envelope.Month(co.DB, types.MonthOf(month.Month))
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	c.JSON(http.StatusOK, EnvelopeMonthResponse{Data: envelopeMonth})
}

// UpdateEnvelope updates data for an envelope
//
//	@Summary		Update envelope
//	@Description	Updates an existing envelope. Only values to be updated need to be specified.
//	@Tags			Envelopes
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	EnvelopeResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string					true	"ID formatted as string"
//	@Param			envelope	body		models.EnvelopeCreate	true	"Envelope"
//	@Router			/v1/envelopes/{envelopeId} [patch]
func (co Controller) UpdateEnvelope(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	envelope, ok := co.getEnvelopeResource(c, p)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.EnvelopeCreate{})
	if err != nil {
		return
	}

	var data models.Envelope
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	if !queryWithRetry(c, co.DB.Model(&envelope).Select("", updateFields...).Updates(data)) {
		return
	}

	envelopeObject, _ := co.getEnvelopeObject(c, envelope.ID)
	c.JSON(http.StatusOK, EnvelopeResponse{Data: envelopeObject})
}

// DeleteEnvelope deletes an envelope
//
//	@Summary		Delete envelope
//	@Description	Deletes an envelope
//	@Tags			Envelopes
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string	true	"ID formatted as string"
//	@Router			/v1/envelopes/{envelopeId} [delete]
func (co Controller) DeleteEnvelope(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	envelope, ok := co.getEnvelopeResource(c, p)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&envelope)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// getEnvelopeResource verifies that the envelope from the URL parameters exists and returns it.
func (co Controller) getEnvelopeResource(c *gin.Context, id uuid.UUID) (models.Envelope, bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "no envelope ID specified")
		return models.Envelope{}, false
	}

	var envelope models.Envelope

	if !queryWithRetry(c, co.DB.Where(&models.Envelope{
		DefaultModel: models.DefaultModel{
			ID: id,
		},
	}).First(&envelope), "No envelope found for the specified ID") {
		return models.Envelope{}, false
	}

	return envelope, true
}

func (co Controller) getEnvelopeObject(c *gin.Context, id uuid.UUID) (Envelope, bool) {
	resource, ok := co.getEnvelopeResource(c, id)
	if !ok {
		return Envelope{}, false
	}

	url := fmt.Sprintf("%s/v1/envelopes/%s", c.GetString("baseURL"), id)

	return Envelope{
		resource,
		EnvelopeLinks{
			Self:         url,
			Allocations:  url + "/allocations",
			Month:        url + "/YYYY-MM",
			Transactions: fmt.Sprintf("%s/v1/transactions?envelope=%s", c.GetString("baseURL"), id),
		},
	}, true
}

func (co Controller) getEnvelopeObjects(c *gin.Context, categoryID uuid.UUID) ([]Envelope, bool) {
	var envelopes []models.Envelope

	if !queryWithRetry(c, co.DB.Where(&models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			CategoryID: categoryID,
		},
	}).Find(&envelopes)) {
		return []Envelope{}, false
	}

	envelopeObjects := make([]Envelope, 0)
	for _, envelope := range envelopes {
		o, _ := co.getEnvelopeObject(c, envelope.ID)
		envelopeObjects = append(envelopeObjects, o)
	}

	return envelopeObjects, true
}
