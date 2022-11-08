package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MonthConfigLinks struct {
	Self     string `json:"self" example:"https://example.com/api/v1/month-configs/61027ebb-ab75-4a49-9e23-a104ddd9ba6b/2017-10"`
	Envelope string `json:"envelope" example:"https://example.com/api/v1/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b"`
}

type MonthConfig struct {
	models.MonthConfig
	Links MonthConfigLinks `json:"links"`
}

type MonthConfigResponse struct {
	Data MonthConfig `json:"data"`
}

type MonthConfigListResponse struct {
	Data []MonthConfig `json:"data"`
}

type MonthConfigQueryFilter struct {
	EnvelopeID string `form:"envelope"`
	Month      string `form:"month"`
}

type MonthConfigFilter struct {
	EnvelopeID uuid.UUID
	Month      time.Time
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
		Month:      month.Month,
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

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs.
// @Tags        MonthConfigs
// @Success     204
// @Router      /v1/month-configs [options]
func (co Controller) OptionsMonthConfigList(c *gin.Context) {
	httputil.OptionsGet(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        MonthConfigs
// @Success     204
// @Failure     400        {object} httperrors.HTTPError
// @Failure     404        {object} httperrors.HTTPError
// @Param       envelopeId path     string true "ID of the Envelope"
// @Param       month      path     string true "The month in YYYY-MM format"
// @Router      /v1/month-configs/{envelopeId}/{month} [options]
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

// @Summary     Get MonthConfig
// @Description Returns configuration for a specific month
// @Tags        MonthConfigs
// @Produce     json
// @Success     200        {object} MonthConfigResponse
// @Failure     400        {object} httperrors.HTTPError
// @Failure     404        {object} httperrors.HTTPError
// @Param       envelopeId path     string true "ID of the Envelope"
// @Param       month      path     string true "The month in YYYY-MM format"
// @Router      /v1/month-configs/{envelopeId}/{month} [get]
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

	_, ok := co.getEnvelopeObject(c, envelopeID)
	if !ok {
		return
	}

	mConfig, ok := co.getMonthConfigResource(c, envelopeID, month.Month)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, MonthConfigResponse{Data: co.getMonthConfigObject(c, mConfig)})
}

// @Summary     List MonthConfigs
// @Description Returns a list of MonthConfigs
// @Tags        MonthConfigs
// @Produce     json
// @Success     200      {object} MonthConfigListResponse
// @Failure     400      {object} httperrors.HTTPError
// @Failure     404      {object} httperrors.HTTPError
// @Failure     500      {object} httperrors.HTTPError
// @Param       envelope query    string false "Filter by name"
// @Param       month    query    string false "Filter by month"
// @Router      /v1/month-configs [get]
func (co Controller) GetMonthConfigs(c *gin.Context) {
	var filter MonthConfigQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the set parameters in the query string
	queryFields := httputil.GetURLFields(c.Request.URL, filter)

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
	mConfigObjects := make([]MonthConfig, 0)

	for _, mConfig := range mConfigs {
		o := co.getMonthConfigObject(c, mConfig)
		mConfigObjects = append(mConfigObjects, o)
	}

	c.JSON(http.StatusOK, MonthConfigListResponse{Data: mConfigObjects})
}

// @Summary     Create MonthConfig
// @Description Creates a new MonthConfig
// @Tags        MonthConfigs
// @Produce     json
// @Success     201         {object} MonthConfigResponse
// @Failure     400         {object} httperrors.HTTPError
// @Failure     404         {object} httperrors.HTTPError
// @Failure     500         {object} httperrors.HTTPError
// @Param       envelopeId  path     string                   true "ID of the Envelope"
// @Param       month       path     string                   true "The month in YYYY-MM format"
// @Param       monthConfig body     models.MonthConfigCreate true "MonthConfig"
// @Router      /v1/month-configs/{envelopeId}/{month} [post]
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
	mConfig.Month = month.Month

	_, ok := co.getEnvelopeResource(c, mConfig.EnvelopeID)
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

	mConfigObject := co.getMonthConfigObject(c, mConfig)
	c.JSON(http.StatusCreated, MonthConfigResponse{Data: mConfigObject})
}

// @Summary     Create MonthConfig
// @Description Creates a new MonthConfig
// @Tags        MonthConfigs
// @Produce     json
// @Success     201         {object} MonthConfigResponse
// @Failure     400         {object} httperrors.HTTPError
// @Failure     500         {object} httperrors.HTTPError
// @Param       envelopeId  path     string                   true "ID of the Envelope"
// @Param       month       path     string                   true "The month in YYYY-MM format"
// @Param       monthConfig body     models.MonthConfigCreate true "MonthConfig"
// @Router      /v1/month-configs/{envelopeId}/{month} [patch]
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

	_, ok := co.getEnvelopeResource(c, envelopeID)
	if !ok {
		return
	}

	mConfig, ok := co.getMonthConfigResource(c, envelopeID, month.Month)
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

	c.JSON(http.StatusOK, MonthConfigResponse{Data: co.getMonthConfigObject(c, mConfig)})
}

// @Summary     Delete MonthConfig
// @Description Deletes configuration settings for a specific month
// @Tags        MonthConfigs
// @Produce     json
// @Success     204
// @Failure     400        {object} httperrors.HTTPError
// @Failure     404        {object} httperrors.HTTPError
// @Param       envelopeId path     string true "ID of the Envelope"
// @Param       month      path     string true "The month in YYYY-MM format"
// @Router      /v1/month-configs/{envelopeId}/{month} [delete]
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

	_, ok := co.getEnvelopeObject(c, envelopeID)
	if !ok {
		return
	}

	mConfig, ok := co.getMonthConfigResource(c, envelopeID, month.Month)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&mConfig)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

func (co Controller) getMonthConfigObject(c *gin.Context, mConfig models.MonthConfig) MonthConfig {
	return MonthConfig{
		mConfig,
		MonthConfigLinks{
			Self:     fmt.Sprintf("%s/v1/month-configs/%s/%04d-%02d", c.GetString("baseURL"), mConfig.EnvelopeID, mConfig.Month.Year(), mConfig.Month.Month()),
			Envelope: fmt.Sprintf("%s/v1/envelopes/%s", c.GetString("baseURL"), mConfig.EnvelopeID),
		},
	}
}

// getMonthConfigResource verifies that the request URI is valid for the transaction and returns it.
func (co Controller) getMonthConfigResource(c *gin.Context, envelopeID uuid.UUID, month time.Time) (models.MonthConfig, bool) {
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
