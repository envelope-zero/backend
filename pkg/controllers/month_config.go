package controllers

import (
	"net/http"
	"strings"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MonthConfigResponse struct {
	Data models.MonthConfig `json:"data"` // Data for the month
}

type MonthConfigListResponse struct {
	Data []models.MonthConfig `json:"data"` // List of month configs
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

	mConfig, ok := co.getMonthConfigResource(c, envelopeID, types.MonthOf(month.Month))
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
//	@Failure		404			{object}	httperrors.HTTPError
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
	if !queryWithRetry(c, co.DB.Where(&models.MonthConfig{
		EnvelopeID: parsed.EnvelopeID,
		Month:      parsed.Month,
	}, queryFields...).Find(&mConfigs)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	if len(mConfigs) == 0 {
		mConfigs = make([]models.MonthConfig, 0)
	}

	c.JSON(http.StatusOK, MonthConfigListResponse{Data: mConfigs})
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
	if err = httputil.BindData(c, &mConfig); err != nil {
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

	c.JSON(http.StatusCreated, MonthConfigResponse{Data: mConfig})
}

// UpdateMonthConfig updates configuration data for a specific envelope and month
//
//	@Summary		Update MonthConfig
//	@Description	Changes settings of an existing MonthConfig
//	@Tags			MonthConfigs
//	@Produce		json
//	@Success		201			{object}	MonthConfigResponse
//	@Failure		400			{object}	httperrors.HTTPError
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

	mConfig, ok := co.getMonthConfigResource(c, envelopeID, types.MonthOf(month.Month))
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.MonthConfigCreate{})
	if err != nil {
		return
	}

	var data models.MonthConfig
	if err = httputil.BindData(c, &data); err != nil {
		return
	}

	if !queryWithRetry(c, co.DB.Model(&mConfig).Select("", updateFields...).Updates(data)) {
		return
	}

	c.JSON(http.StatusOK, MonthConfigResponse{Data: mConfig})
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

	mConfig, ok := co.getMonthConfigResource(c, envelopeID, types.MonthOf(month.Month))
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&mConfig)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// getMonthConfigResource verifies that the request URI is valid for the transaction and returns it.
func (co Controller) getMonthConfigResource(c *gin.Context, envelopeID uuid.UUID, month types.Month) (models.MonthConfig, bool) {
	if envelopeID == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "no envelope ID specified")
		return models.MonthConfig{}, false
	}

	var mConfig models.MonthConfig

	if !queryWithRetry(c, co.DB.First(&mConfig, &models.MonthConfig{
		EnvelopeID: envelopeID,
		Month:      month,
	}), "No MonthConfig found for the Envelope and month specified") {
		return models.MonthConfig{}, false
	}

	return mConfig, true
}
