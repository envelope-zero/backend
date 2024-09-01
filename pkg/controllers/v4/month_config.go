package v4

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
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
// @Failure		400		{object}	httpError
// @Param			id		path		URIID		true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			month	path		URIMonth	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/envelopes/{id}/{month} [options]
func OptionsMonthConfigDetail(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
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
// @Param			id		path		URIID		true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			month	path		URIMonth	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/envelopes/{id}/{month} [get]
func GetMonthConfig(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	err = models.DB.First(&models.Envelope{}, uri.ID).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	mConfig, err := getMonthConfigModel(uri.ID.UUID, types.MonthOf(month.Month))
	var data MonthConfig
	if err != nil {
		// If there is no MonthConfig in the database, return one with the zero values
		if errors.Is(err, models.ErrResourceNotFound) {
			data = newMonthConfig(c, models.MonthConfig{
				EnvelopeID: uri.ID.UUID,
				Month:      types.MonthOf(month.Month),
			})
			c.JSON(http.StatusOK, MonthConfigResponse{Data: &data})
			return
		}

		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
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
// @Param			id			path		URIID				true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			month		path		URIMonth			true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			monthConfig	body		MonthConfigEditable	true	"MonthConfig"
// @Router			/v4/envelopes/{id}/{month} [patch]
func UpdateMonthConfig(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	err = models.DB.First(&models.Envelope{}, uri.ID).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, MonthConfigEditable{})
	if err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var data MonthConfigEditable
	err = httputil.BindData(c, &data)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	m, err := getMonthConfigModel(uri.ID.UUID, types.MonthOf(month.Month))

	// If no Month Config exists yet, create one
	if err != nil && errors.Is(err, models.ErrResourceNotFound) {
		data.EnvelopeID = uri.ID.UUID
		data.Month = types.Month(month.Month)

		model := data.model()
		e := models.DB.Create(&model).Error

		if e != nil {
			s := err.Error()
			c.JSON(status(err), MonthConfigResponse{
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
	if err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	// Perform the actual update
	err = models.DB.Model(&m).Select("", updateFields...).Updates(data.model()).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), MonthConfigResponse{
			Error: &s,
		})
		return
	}

	apiResource := newMonthConfig(c, m)
	c.JSON(http.StatusOK, MonthConfigResponse{Data: &apiResource})
}
