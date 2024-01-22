package v4

import (
	"net/http"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

// RegisterEnvelopeRoutes registers the routes for envelopes with
// the RouterGroup that is passed.
func RegisterEnvelopeRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsEnvelopeList)
		r.GET("", GetEnvelopes)
		r.POST("", CreateEnvelopes)
	}

	// Envelope with ID
	{
		r.OPTIONS("/:id", OptionsEnvelopeDetail)
		r.GET("/:id", GetEnvelope)
		r.PATCH("/:id", UpdateEnvelope)
		r.DELETE("/:id", DeleteEnvelope)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Envelopes
// @Success		204
// @Router			/v4/envelopes [options]
func OptionsEnvelopeList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Envelopes
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/envelopes/{id} [options]
func OptionsEnvelopeDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.First(&models.Envelope{}, id).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// @Summary		Create envelope
// @Description	Creates a new envelope
// @Tags			Envelopes
// @Produce		json
// @Success		201			{object}	EnvelopeCreateResponse
// @Failure		400			{object}	EnvelopeCreateResponse
// @Failure		404			{object}	EnvelopeCreateResponse
// @Failure		500			{object}	EnvelopeCreateResponse
// @Param			envelope	body		[]v4.EnvelopeEditable	true	"Envelopes"
// @Router			/v4/envelopes [post]
func CreateEnvelopes(c *gin.Context) {
	var envelopes []EnvelopeEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &envelopes)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), EnvelopeCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := EnvelopeCreateResponse{}

	for _, editable := range envelopes {
		envelope := editable.model()
		err = models.DB.Create(&envelope).Error
		if err != nil {
			status = r.appendError(err, status)
			continue
		}

		data := newEnvelope(c, envelope)
		r.Data = append(r.Data, EnvelopeResponse{Data: &data})
	}

	c.JSON(status, r)
}

// @Summary		Get envelopes
// @Description	Returns a list of envelopes
// @Tags			Envelopes
// @Produce		json
// @Success		200	{object}	EnvelopeListResponse
// @Failure		400	{object}	EnvelopeListResponse
// @Failure		500	{object}	EnvelopeListResponse
// @Router			/v4/envelopes [get]
// @Param			name		query	string	false	"Filter by name"
// @Param			note		query	string	false	"Filter by note"
// @Param			category	query	string	false	"Filter by category ID"
// @Param			archived	query	bool	false	"Is the envelope archived?"
// @Param			search		query	string	false	"Search for this text in name and note"
// @Param			offset		query	uint	false	"The offset of the first Envelope returned. Defaults to 0."
// @Param			limit		query	int		false	"Maximum number of Envelopes to return. Defaults to 50."
func GetEnvelopes(c *gin.Context) {
	var filter EnvelopeQueryFilter

	// The filters contain only strings, so this will always succeed
	_ = c.Bind(&filter)

	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	model, err := filter.model()
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeListResponse{
			Error: &s,
		})
		return
	}

	q := models.DB.
		Order("name ASC").
		Where(&model, queryFields...)

	q = stringFilters(models.DB, q, setFields, filter.Name, filter.Note, filter.Search)

	// Set the offset. Does not need checking since the default is 0
	q = q.Offset(int(filter.Offset))

	// Default to 50 Accounts and set the limit
	limit := 50
	if slices.Contains(setFields, "Limit") {
		limit = filter.Limit
	}
	q = q.Limit(limit)

	var envelopes []models.Envelope
	err = q.Find(&envelopes).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = q.Limit(-1).Offset(-1).Count(&count).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), EnvelopeListResponse{
			Error: &e,
		})
		return
	}

	data := make([]Envelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		data = append(data, newEnvelope(c, envelope))
	}

	c.JSON(http.StatusOK, EnvelopeListResponse{
		Data: data,
		Pagination: &Pagination{
			Count:  len(data),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// @Summary		Get Envelope
// @Description	Returns a specific Envelope
// @Tags			Envelopes
// @Produce		json
// @Success		200	{object}	EnvelopeResponse
// @Failure		400	{object}	EnvelopeResponse
// @Failure		404	{object}	EnvelopeResponse
// @Failure		500	{object}	EnvelopeResponse
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/envelopes/{id} [get]
func GetEnvelope(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeResponse{
			Error: &s,
		})
		return
	}

	var envelope models.Envelope
	err = models.DB.First(&envelope, id).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeResponse{
			Error: &s,
		})
		return
	}

	data := newEnvelope(c, envelope)
	c.JSON(http.StatusOK, EnvelopeResponse{Data: &data})
}

// @Summary		Update envelope
// @Description	Updates an existing envelope. Only values to be updated need to be specified.
// @Tags			Envelopes
// @Accept			json
// @Produce		json
// @Success		200			{object}	EnvelopeResponse
// @Failure		400			{object}	EnvelopeResponse
// @Failure		404			{object}	EnvelopeResponse
// @Failure		500			{object}	EnvelopeResponse
// @Param			id			path		string				true	"ID formatted as string"
// @Param			envelope	body		v4.EnvelopeEditable	true	"Envelope"
// @Router			/v4/envelopes/{id} [patch]
func UpdateEnvelope(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeResponse{
			Error: &s,
		})
		return
	}

	var envelope models.Envelope
	err = models.DB.First(&envelope, id).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, EnvelopeEditable{})
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeResponse{
			Error: &s,
		})
		return
	}

	var data EnvelopeEditable
	err = httputil.BindData(c, &data)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeResponse{
			Error: &s,
		})
		return
	}

	err = models.DB.Model(&envelope).Select("", updateFields...).Updates(data.model()).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), EnvelopeResponse{
			Error: &s,
		})
		return
	}

	apiResource := newEnvelope(c, envelope)
	c.JSON(http.StatusOK, EnvelopeResponse{Data: &apiResource})
}

// @Summary		Delete envelope
// @Description	Deletes an envelope
// @Tags			Envelopes
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/envelopes/{id} [delete]
func DeleteEnvelope(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var envelope models.Envelope
	err = models.DB.First(&envelope, id).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.Delete(&envelope).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
