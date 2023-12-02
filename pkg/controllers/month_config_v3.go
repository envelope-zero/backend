package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MonthConfigV3 struct {
	models.MonthConfig
	OverspendMode string `json:"overspendMode,omitempty"` // Ignore this. It is here to override the OverspendMode from models.MonthConfigCreate and will be removed with 4.0.0
	Links         struct {
		Self     string `json:"self" example:"https://example.com/api/v3/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b/2017-10"` // The Month Config itself
		Envelope string `json:"envelope" example:"https://example.com/api/v3/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b"`     // The Envelope this config belongs to
	} `json:"links"`
}

func (m *MonthConfigV3) links(c *gin.Context) {
	url := c.GetString(string(database.ContextURL))

	m.Links.Self = fmt.Sprintf("%s/v3/envelopes/%s/%s", url, m.EnvelopeID, m.Month)
	m.Links.Envelope = fmt.Sprintf("%s/v3/envelopes/%s", url, m.EnvelopeID)
}

func (co Controller) getMonthConfigV3(c *gin.Context, id uuid.UUID, month types.Month) (MonthConfigV3, httperrors.Error) {
	m, err := co.getMonthConfigModelV3(c, id, month)
	if !err.Nil() {
		return MonthConfigV3{}, err
	}

	r := MonthConfigV3{
		MonthConfig: m,
	}

	r.links(c)
	return r, httperrors.Error{}
}

func (co Controller) getMonthConfigModelV3(c *gin.Context, id uuid.UUID, month types.Month) (models.MonthConfig, httperrors.Error) {
	var m models.MonthConfig

	err := query(c, co.DB.First(&m, &models.MonthConfig{
		EnvelopeID: id,
		Month:      month,
	}))

	if !err.Nil() {
		return models.MonthConfig{}, err
	}

	return m, httperrors.Error{}
}

type MonthConfigResponseV3 struct {
	Data  *MonthConfigV3 `json:"data"`                                                          // Config for the month
	Error *string        `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type MonthConfigListResponseV3 struct {
	Data       []MonthConfigV3 `json:"data"`                                                          // List of Month Configs
	Error      *string         `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination     `json:"pagination"`                                                    // Pagination information
}

type MonthConfigQueryFilterV3 struct {
	EnvelopeID string `form:"envelope"`                   // By ID of the envelope
	Month      string `form:"month"`                      // By month
	Offset     uint   `form:"offset" filterField:"false"` // The offset of the first Month Config returned. Defaults to 0.
	Limit      int    `form:"limit" filterField:"false"`  // Maximum number of Month Configs to return. Defaults to 50.
}

func (m MonthConfigQueryFilterV3) Parse(c *gin.Context) (MonthConfigFilter, httperrors.Error) {
	envelopeID, err := httputil.UUIDFromString(m.EnvelopeID)
	if !err.Nil() {
		return MonthConfigFilter{}, err
	}

	var month QueryMonth
	if err := c.Bind(&month); err != nil {
		return MonthConfigFilter{}, httperrors.Parse(c, err)
	}

	return MonthConfigFilter{
		EnvelopeID: envelopeID,
		Month:      types.MonthOf(month.Month),
	}, httperrors.Error{}
}

// RegisterMonthConfigRoutesV3 registers the routes for transactions with
// the RouterGroup that is passed.
func (co Controller) RegisterMonthConfigRoutesV3(r *gin.RouterGroup) {
	r.OPTIONS("/:id/:month", co.OptionsMonthConfigDetailV3)
	r.GET("/:id/:month", co.GetMonthConfigV3)
	r.PATCH("/:id/:month", co.UpdateMonthConfigV3)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Envelopes
// @Success		204
// @Failure		400		{object}	httperrors.HTTPError
// @Param			id		path		string	true	"ID of the Envelope"
// @Param			month	path		string	true	"The month in YYYY-MM format"
// @Router			/v3/envelopes/{id}/{month} [options]
func (co Controller) OptionsMonthConfigDetailV3(c *gin.Context) {
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
// @Success		200		{object}	MonthConfigResponseV3
// @Failure		400		{object}	MonthConfigResponseV3
// @Failure		404		{object}	MonthConfigResponseV3
// @Failure		500		{object}	MonthConfigResponseV3
// @Param			id		path		string	true	"ID of the Envelope"
// @Param			month	path		string	true	"The month in YYYY-MM format"
// @Router			/v3/envelopes/{id}/{month} [get]
func (co Controller) GetMonthConfigV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
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

	_, err = getResourceByID[models.Envelope](c, co, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	mConfig, err := co.getMonthConfigV3(c, id, types.MonthOf(month.Month))
	if !err.Nil() {
		// If there is no MonthConfig in the database, return one with the zero values
		if errors.Is(err.Err, httperrors.ErrNoResource) {
			mConfig := MonthConfigV3{
				MonthConfig: models.MonthConfig{
					EnvelopeID: id,
					Month:      types.MonthOf(month.Month),
				},
			}
			mConfig.links(c)

			c.JSON(http.StatusOK, MonthConfigResponseV3{Data: &mConfig})
			return
		}

		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, MonthConfigResponseV3{Data: &mConfig})
}

// @Summary		Update MonthConfig
// @Description	Changes configuration for a Month. If there is no configuration for the month yet, this endpoint transparently creates a configuration resource.
// @Tags			Envelopes
// @Produce		json
// @Success		201			{object}	MonthConfigResponseV3
// @Failure		400			{object}	MonthConfigResponseV3
// @Failure		404			{object}	MonthConfigResponseV3
// @Failure		500			{object}	MonthConfigResponseV3
// @Param			id			path		string						true	"ID of the Envelope"
// @Param			month		path		string						true	"The month in YYYY-MM format"
// @Param			monthConfig	body		models.MonthConfigCreate	true	"MonthConfig"
// @Router			/v3/envelopes/{id}/{month} [patch]
func (co Controller) UpdateMonthConfigV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	_, err = getResourceByID[models.Envelope](c, co, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	m, err := co.getMonthConfigModelV3(c, id, types.MonthOf(month.Month))
	if !err.Nil() {
		// If no Month Config exists yet, create one
		// If the error is another error, return it to the user
		if errors.Is(err.Err, httperrors.ErrNoResource) {
			e := co.DB.Create(&models.MonthConfig{
				Month:      types.MonthOf(month.Month),
				EnvelopeID: id,
			}).Error

			if e != nil {
				err = httperrors.Parse(c, err)
				s := e.Error()
				c.JSON(err.Status, MonthConfigResponseV3{
					Error: &s,
				})
			}

			m, err = co.getMonthConfigModelV3(c, id, types.MonthOf(month.Month))
			if !err.Nil() {
				s := err.Error()
				c.JSON(err.Status, MonthConfigResponseV3{
					Error: &s,
				})
				return
			}
		} else {
			s := err.Error()
			c.JSON(err.Status, MonthConfigResponseV3{
				Error: &s,
			})
			return
		}
	}

	updateFields, err := httputil.GetBodyFields(c, models.MonthConfigCreate{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	var data models.MonthConfig
	err = httputil.BindData(c, &data.MonthConfigCreate)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	err = query(c, co.DB.Model(&m).Select("", updateFields...).Updates(data))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	o, err := co.getMonthConfigV3(c, m.EnvelopeID, m.Month)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, MonthConfigResponseV3{Data: &o})
}
