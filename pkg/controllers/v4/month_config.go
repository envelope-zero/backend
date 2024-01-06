package v4

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
)

// RegisterMonthConfigRoutes registers the routes for transactions with
// the RouterGroup that is passed.
func RegisterMonthConfigRoutes(r *gin.RouterGroup) {
	r.OPTIONS("/:id/:month", OptionsMonthConfigDetail)
	r.GET("/:id/:month", GetMonthConfig)
	r.PATCH("/:id/:month", UpdateMonthConfig)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Envelopes
// @Success		204
// @Failure		400		{object}	httperrors.HTTPError
// @Param			id		path		string	true	"ID of the Envelope"
// @Param			month	path		string	true	"The month in YYYY-MM format"
// @Router			/v4/envelopes/{id}/{month} [options]
func OptionsMonthConfigDetail(c *gin.Context) {
	_, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		e := httperrors.Parse(c, err)
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	httputil.OptionsGetPatch(c)
}

// @Summary		Get MonthConfig
// @Description	Returns configuration for a specific month
// @Tags			Envelopes
// @Produce		json
// @Success		200		{object}	MonthConfigResponse
// @Failure		400		{object}	MonthConfigResponse
// @Failure		404		{object}	MonthConfigResponse
// @Failure		500		{object}	MonthConfigResponse
// @Param			id		path		string	true	"ID of the Envelope"
// @Param			month	path		string	true	"The month in YYYY-MM format"
// @Router			/v4/envelopes/{id}/{month} [get]
func GetMonthConfig(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		e := httperrors.Parse(c, err)
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	_, err = getModelByID[models.Envelope](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	mConfig, err := getMonthConfigModel(c, id, types.MonthOf(month.Month))
	var data MonthConfig
	if !err.Nil() {
		// If there is no MonthConfig in the database, return one with the zero values
		if errors.Is(err.Err, httperrors.ErrNoResource) {
			data = newMonthConfig(c, models.MonthConfig{
				EnvelopeID: id,
				Month:      types.MonthOf(month.Month),
			})
			c.JSON(http.StatusOK, MonthConfigResponse{Data: &data})
			return
		}

		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	data = newMonthConfig(c, mConfig)
	c.JSON(http.StatusOK, MonthConfigResponse{Data: &data})
}

// @Summary		Update MonthConfig
// @Description	Changes configuration for a Month. If there is no configuration for the month yet, this endpoint transparently creates a configuration resource.
// @Tags			Envelopes
// @Produce		json
// @Success		201			{object}	MonthConfigResponse
// @Failure		400			{object}	MonthConfigResponse
// @Failure		404			{object}	MonthConfigResponse
// @Failure		500			{object}	MonthConfigResponse
// @Param			id			path		string				true	"ID of the Envelope"
// @Param			month		path		string				true	"The month in YYYY-MM format"
// @Param			monthConfig	body		MonthConfigEditable	true	"MonthConfig"
// @Router			/v4/envelopes/{id}/{month} [patch]
func UpdateMonthConfig(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	_, err = getModelByID[models.Envelope](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, MonthConfigEditable{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var data MonthConfigEditable
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	m, err := getMonthConfigModel(c, id, types.MonthOf(month.Month))

	// If no Month Config exists yet, create one
	if !err.Nil() && errors.Is(err.Err, httperrors.ErrNoResource) {
		data.EnvelopeID = id
		data.Month = types.Month(month.Month)

		model := data.model()
		e := models.DB.Create(&model).Error

		if e != nil {
			err = httperrors.Parse(c, err)
			s := e.Error()
			c.JSON(err.Status, MonthConfigResponse{
				Error: &s,
			})
		}

		apiResource := newMonthConfig(c, model)
		c.JSON(http.StatusOK, MonthConfigResponse{
			Data: &apiResource,
		})
		return
	}

	// Handle all other errors
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	// Perform the actual update
	err = query(c, models.DB.Model(&m).Select("", updateFields...).Updates(data.model()))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	apiResource := newMonthConfig(c, m)
	c.JSON(http.StatusOK, MonthConfigResponse{Data: &apiResource})
}
