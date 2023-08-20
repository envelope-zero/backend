package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EnvelopeListResponse struct {
	Data []models.Envelope `json:"data"` // List of Envelopes
}

type EnvelopeResponse struct {
	Data models.Envelope `json:"data"` // Data for the Envelope
}

type EnvelopeMonthResponse struct {
	Data models.EnvelopeMonth `json:"data"` // Data for the month for the envelope
}

type EnvelopeQueryFilter struct {
	Name       string `form:"name" filterField:"false"`   // By name
	CategoryID string `form:"category"`                   // By the ID of the category
	Note       string `form:"note" filterField:"false"`   // By the note
	Hidden     bool   `form:"hidden"`                     // Is the envelope archived?
	Search     string `form:"search" filterField:"false"` // By string in name or note
}

func (f EnvelopeQueryFilter) ToCreate(c *gin.Context) (models.EnvelopeCreate, bool) {
	categoryID, ok := httputil.UUIDFromString(c, f.CategoryID)
	if !ok {
		return models.EnvelopeCreate{}, false
	}

	return models.EnvelopeCreate{
		CategoryID: categoryID,
		Hidden:     f.Hidden,
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
	id, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, id)
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

	_, ok := getResourceByIDAndHandleErrors[models.Category](c, co, envelope.CategoryID)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Create(&envelope)) {
		return
	}

	c.JSON(http.StatusCreated, EnvelopeResponse{Data: envelope})
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
//	@Param			hidden		query	bool	false	"Is the envelope hidden?"
//	@Param			search		query	string	false	"Search for this text in name and note"
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

	query = stringFilters(co.DB, query, setFields, filter.Name, filter.Note, filter.Search)

	var envelopes []models.Envelope
	if !queryWithRetry(c, query.Find(&envelopes)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	if len(envelopes) == 0 {
		envelopes = make([]models.Envelope, 0)
	}

	c.JSON(http.StatusOK, EnvelopeListResponse{Data: envelopes})
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
	id, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	envelopeObject, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, id)
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
	id, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	envelope, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, id)
	if !ok {
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
	id, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	envelope, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, id)
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

	c.JSON(http.StatusOK, EnvelopeResponse{Data: envelope})
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
	id, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	envelope, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, id)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&envelope)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
