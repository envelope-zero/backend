package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MonthConfig struct {
	models.MonthConfig
	Links struct {
		Self     string `json:"self" example:"https://example.com/api/v1/month-configs/61027ebb-ab75-4a49-9e23-a104ddd9ba6b/2017-10"` // The month config itself
		Envelope string `json:"envelope" example:"https://example.com/api/v1/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b"`         // The envelope this config belongs to
	} `json:"links"`
}

func (m *MonthConfig) links(c *gin.Context) {
	url := c.GetString(string(database.ContextURL))

	m.Links.Self = fmt.Sprintf("%s/v1/month-configs/%s/%s", url, m.EnvelopeID, m.Month)
	m.Links.Envelope = fmt.Sprintf("%s/v1/envelopes/%s", url, m.EnvelopeID)
}

func (co Controller) getMonthConfig(c *gin.Context, id uuid.UUID, month types.Month) (MonthConfig, bool) {
	m, ok := co.getMonthConfigModel(c, id, month)
	if !ok {
		return MonthConfig{}, false
	}

	r := MonthConfig{
		MonthConfig: m,
	}

	r.links(c)
	return r, true
}

func (co Controller) getMonthConfigModel(c *gin.Context, id uuid.UUID, month types.Month) (models.MonthConfig, bool) {
	var m models.MonthConfig

	if !queryAndHandleErrors(c, co.DB.First(&m, &models.MonthConfig{
		EnvelopeID: id,
		Month:      month,
	}), "No MonthConfig found for the Envelope and month specified") {
		return models.MonthConfig{}, false
	}

	return m, true
}

type MonthConfigResponse struct {
	Data MonthConfig `json:"data"` // Data for the month
}

type MonthConfigListResponse struct {
	Data []MonthConfig `json:"data"` // List of month configs
}

type MonthConfigQueryFilter struct {
	EnvelopeID string `form:"envelope"` // By ID of the envelope
	Month      string `form:"month"`    // By month
}

type MonthConfigFilter struct {
	EnvelopeID uuid.UUID
	Month      types.Month
}

func (m MonthConfigQueryFilter) Parse(c *gin.Context) (MonthConfigFilter, bool) {
	envelopeID, ok := httputil.UUIDFromString(c, m.EnvelopeID)
	if !ok {
		return MonthConfigFilter{}, false
	}

	var month QueryMonth
	if err := c.Bind(&month); err != nil {
		httperrors.Handler(c, err)
		return MonthConfigFilter{}, false
	}

	return MonthConfigFilter{
		EnvelopeID: envelopeID,
		Month:      types.MonthOf(month.Month),
	}, true
}

// RegisterMonthConfigRoutes registers the routes for transactions with
// the RouterGroup that is passed.
func (co Controller) RegisterMonthConfigRoutes(r *gin.RouterGroup) {
	r.OPTIONS("", co.OptionsMonthConfigList)
	r.GET("", co.GetMonthConfigs)

	r.OPTIONS("/:envelopeId/:month", co.OptionsMonthConfigDetail)
	r.GET("/:envelopeId/:month", co.GetMonthConfig)
	r.POST("/:envelopeId/:month", co.CreateMonthConfig)
	r.PATCH("/:envelopeId/:month", co.UpdateMonthConfig)
	r.DELETE("/:envelopeId/:month", co.DeleteMonthConfig)
}

// OptionsMonthConfigList returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs.
//	@Tags			MonthConfigs
//	@Success		204
//	@Router			/v1/month-configs [options]
func (co Controller) OptionsMonthConfigList(c *gin.Context) {
	httputil.OptionsGet(c)
}

// OptionsMonthConfigDetail returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			MonthConfigs
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string	true	"ID of the Envelope"
//	@Param			month		path		string	true	"The month in YYYY-MM format"
//	@Router			/v1/month-configs/{envelopeId}/{month} [options]
func (co Controller) OptionsMonthConfigDetail(c *gin.Context) {
	_, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	httputil.OptionsGetPostPatchDelete(c)
}

// GetMonthConfig returns config for a specific envelope and month
//
//	@Summary		Get MonthConfig
//	@Description	Returns configuration for a specific month
//	@Tags			MonthConfigs
//	@Produce		json
//	@Success		200			{object}	MonthConfigResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string	true	"ID of the Envelope"
//	@Param			month		path		string	true	"The month in YYYY-MM format"
//	@Router			/v1/month-configs/{envelopeId}/{month} [get]
func (co Controller) GetMonthConfig(c *gin.Context) {
	envelopeID, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, envelopeID)
	if !ok {
		return
	}

	mConfig, ok := co.getMonthConfig(c, envelopeID, types.MonthOf(month.Month))
	if !ok {
		return
	}

	c.JSON(http.StatusOK, MonthConfigResponse{Data: mConfig})
}

// GetMonthConfigs returns all month configs filtered by the query parameters
//
//	@Summary		List MonthConfigs
//	@Description	Returns a list of MonthConfigs
//	@Tags			MonthConfigs
//	@Produce		json
//	@Success		200			{object}	MonthConfigListResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelope	query		string	false	"Filter by name"
//	@Param			month		query		string	false	"Filter by month"
//	@Router			/v1/month-configs [get]
func (co Controller) GetMonthConfigs(c *gin.Context) {
	var filter MonthConfigQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the set parameters in the query string
	queryFields, _ := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Filter struct
	parsed, ok := filter.Parse(c)
	if !ok {
		return
	}

	var mConfigs []models.MonthConfig
	if !queryAndHandleErrors(c, co.DB.Where(&models.MonthConfig{
		EnvelopeID: parsed.EnvelopeID,
		Month:      parsed.Month,
	}, queryFields...).Find(&mConfigs)) {
		return
	}

	r := make([]MonthConfig, 0)
	for _, m := range mConfigs {
		o, ok := co.getMonthConfig(c, m.EnvelopeID, m.Month)
		if !ok {
			return
		}
		r = append(r, o)
	}

	c.JSON(http.StatusOK, MonthConfigListResponse{Data: r})
}

// CreateMonthConfig creates a new month config
//
//	@Summary		Create MonthConfig
//	@Description	Creates a new MonthConfig
//	@Tags			MonthConfigs
//	@Produce		json
//	@Success		201			{object}	MonthConfigResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string						true	"ID of the Envelope"
//	@Param			month		path		string						true	"The month in YYYY-MM format"
//	@Param			monthConfig	body		models.MonthConfigCreate	true	"MonthConfig"
//	@Router			/v1/month-configs/{envelopeId}/{month} [post]
func (co Controller) CreateMonthConfig(c *gin.Context) {
	envelopeID, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	var mConfig models.MonthConfig
	if err = httputil.BindData(c, &mConfig.MonthConfigCreate); err != nil {
		return
	}

	// Set config to path parameters
	mConfig.EnvelopeID = envelopeID
	mConfig.Month = types.MonthOf(month.Month)

	_, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, mConfig.EnvelopeID)
	if !ok {
		return
	}

	err = co.DB.Create(&mConfig).Error
	if err != nil {
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
			httperrors.Handler(c, err)
			return
		}

		httperrors.New(c, http.StatusBadRequest, "Cannot create MonthConfig for Envelope with ID %s and month %s as it already exists", mConfig.EnvelopeID, mConfig.Month)
		return
	}

	r, ok := co.getMonthConfig(c, mConfig.EnvelopeID, mConfig.Month)
	if !ok {
		return
	}

	c.JSON(http.StatusCreated, MonthConfigResponse{Data: r})
}

// UpdateMonthConfig updates configuration data for a specific envelope and month
//
//	@Summary		Update MonthConfig
//	@Description	Changes settings of an existing MonthConfig
//	@Tags			MonthConfigs
//	@Produce		json
//	@Success		201			{object}	MonthConfigResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string						true	"ID of the Envelope"
//	@Param			month		path		string						true	"The month in YYYY-MM format"
//	@Param			monthConfig	body		models.MonthConfigCreate	true	"MonthConfig"
//	@Router			/v1/month-configs/{envelopeId}/{month} [patch]
func (co Controller) UpdateMonthConfig(c *gin.Context) {
	envelopeID, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, envelopeID)
	if !ok {
		return
	}

	m, ok := co.getMonthConfigModel(c, envelopeID, types.MonthOf(month.Month))
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.MonthConfigCreate{})
	if err != nil {
		return
	}

	var data models.MonthConfig
	if err = httputil.BindData(c, &data.MonthConfigCreate); err != nil {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Model(&m).Select("", updateFields...).Updates(data)) {
		return
	}

	o, ok := co.getMonthConfig(c, m.EnvelopeID, m.Month)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, MonthConfigResponse{Data: o})
}

// DeleteMonthConfig deletes configuration data for a specific envelope and month
//
//	@Summary		Delete MonthConfig
//	@Description	Deletes configuration settings for a specific month
//	@Tags			MonthConfigs
//	@Produce		json
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			envelopeId	path		string	true	"ID of the Envelope"
//	@Param			month		path		string	true	"The month in YYYY-MM format"
//	@Router			/v1/month-configs/{envelopeId}/{month} [delete]
func (co Controller) DeleteMonthConfig(c *gin.Context) {
	envelopeID, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, envelopeID)
	if !ok {
		return
	}

	m, ok := co.getMonthConfigModel(c, envelopeID, types.MonthOf(month.Month))
	if !ok {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Delete(&m)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
