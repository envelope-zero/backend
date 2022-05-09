package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type EnvelopeLinks struct {
	Allocations string `json:"allocations" example:"https://example.com/api/v1/budgets/2/categories/5/envelopes/1/allocations"`
	Month       string `json:"month" example:"https://example.com/api/v1/budgets/2/categories/5/envelopes/1/2019-03"`
}

type EnvelopeResponse struct {
	Data  models.Envelope `json:"data"`
	Links EnvelopeLinks   `json:"links"`
}

type EnvelopeListResponse struct {
	Data []models.Envelope `json:"data"`
}

type EnvelopeMonthResponse struct {
	Data models.EnvelopeMonth `json:"data"`
}

// RegisterEnvelopeRoutes registers the routes for envelopes with
// the RouterGroup that is passed.
func RegisterEnvelopeRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsEnvelopeList)
		r.GET("", GetEnvelopes)
		r.POST("", CreateEnvelope)
	}

	// Envelope with ID
	{
		r.OPTIONS("/:envelopeId", OptionsEnvelopeDetail)
		r.GET("/:envelopeId", GetEnvelope)
		r.GET("/:envelopeId/:month", GetEnvelopeMonth)
		r.PATCH("/:envelopeId", UpdateEnvelope)
		r.DELETE("/:envelopeId", DeleteEnvelope)
	}

	RegisterAllocationRoutes(r.Group("/:envelopeId/allocations"))
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Envelopes
// @Success      204
// @Param        budgetId    path  uint64  true  "ID of the budget"
// @Param        categoryId  path  uint64  true  "ID of the category"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes [options]
func OptionsEnvelopeList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Envelopes
// @Success      204
// @Param        budgetId    path  uint64  true  "ID of the budget"
// @Param        categoryId  path  uint64  true  "ID of the category"
// @Param        envelopeId  path  uint64  true  "ID of the envelope"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId} [options]
func OptionsEnvelopeDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// @Summary      Create envelope
// @Description  Create a new envelope for a specific category
// @Tags         Envelopes
// @Produce      json
// @Success      201  {object}  EnvelopeResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId    path      uint64                 true  "ID of the budget"
// @Param        categoryId  path      uint64  true  "ID of the category"
// @Param        envelope    body      models.EnvelopeCreate  true  "Envelope"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes [post]
func CreateEnvelope(c *gin.Context) {
	var data models.Envelope

	err := httputil.BindData(c, &data)
	if err != nil {
		return
	}

	data.CategoryID, err = httputil.ParseID(c, "categoryId")
	if err != nil {
		return
	}
	models.DB.Create(&data)
	c.JSON(http.StatusCreated, EnvelopeResponse{Data: data})
}

// @Summary      Get all envelopes for a category
// @Description  Returns the full list of all envelopes for a specific category
// @Tags         Envelopes
// @Produce      json
// @Success      200  {object}  EnvelopeListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId    path      uint64  true  "ID of the budget"
// @Param        categoryId  path      uint64                 true  "ID of the category"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes [get]
func GetEnvelopes(c *gin.Context) {
	var envelopes []models.Envelope

	// Check if the category exists at all
	category, err := getCategoryResource(c)
	if err != nil {
		return
	}

	models.DB.Where(&models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			CategoryID: category.ID,
		},
	}).Find(&envelopes)

	c.JSON(http.StatusOK, EnvelopeListResponse{Data: envelopes})
}

// @Summary      Get envelope
// @Description  Returns an envelope by its ID
// @Tags         Envelopes
// @Produce      json
// @Success      200  {object}  EnvelopeResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId    path      uint64  true  "ID of the budget"
// @Param        categoryId  path      uint64  true  "ID of the category"
// @Param        envelopeId  path      uint64                 true  "ID of the envelope"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId} [get]
func GetEnvelope(c *gin.Context) {
	_, err := getEnvelopeResource(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, newEnevlopeResponse(c))
}

// @Summary      Get Envelope month data
// @Description  Returns data about an envelope for a for a specific month
// @Tags         Envelopes
// @Produce      json
// @Success      200  {object}  EnvelopeMonthResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId    path      uint64  true  "ID of the budget"
// @Param        budgetId    path      uint64  true  "ID of the budget"
// @Param        categoryId  path      uint64  true  "ID of the category"
// @Param        envelopeId  path      uint64  true  "ID of the envelope"
// @Param        month       path      string  true  "The month in YYYY-MM format"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId}/{month} [get]
func GetEnvelopeMonth(c *gin.Context) {
	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		return
	}

	envelope, err := getEnvelopeResource(c)
	if err != nil {
		return
	}

	if month.Month.IsZero() {
		httputil.NewError(c, http.StatusBadRequest, errors.New("You cannot request data for no month"))
		return
	}

	c.JSON(http.StatusOK, EnvelopeMonthResponse{Data: envelope.Month(month.Month)})
}

// @Summary      Update an envelope
// @Description  Update an existing envelope. Only values to be updated need to be specified.
// @Tags         Envelopes
// @Accept       json
// @Produce      json
// @Success      200  {object}  EnvelopeResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId    path      uint64                 true  "ID of the budget"
// @Param        categoryId  path      uint64                 true  "ID of the category"
// @Param        envelopeId  path      uint64  true  "ID of the envelope"
// @Param        envelope    body      models.EnvelopeCreate  true  "Envelope"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId} [patch]
func UpdateEnvelope(c *gin.Context) {
	var envelope models.Envelope

	err := models.DB.First(&envelope, c.Param("envelopeId")).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return
	}

	var data models.Envelope
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	models.DB.Model(&envelope).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": envelope})
}

// @Summary      Delete an envelope
// @Description  Deletes an existing envelope
// @Tags         Envelopes
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId    path      uint64  true  "ID of the budget"
// @Param        categoryId  path      uint64  true  "ID of the category"
// @Param        envelopeId  path      uint64  true  "ID of the envelope"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId} [delete]
func DeleteEnvelope(c *gin.Context) {
	envelope, err := getEnvelopeResource(c)
	if err != nil {
		return
	}

	models.DB.Delete(&envelope)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getEnvelopeResource verifies that the envelope from the URL parameters exists and returns it.
func getEnvelopeResource(c *gin.Context) (models.Envelope, error) {
	var envelope models.Envelope

	envelopeID, err := httputil.ParseID(c, "envelopeId")
	if err != nil {
		return models.Envelope{}, err
	}

	category, err := getCategoryResource(c)
	if err != nil {
		return models.Envelope{}, err
	}

	err = models.DB.Where(&models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			CategoryID: category.ID,
		},
		Model: models.Model{
			ID: envelopeID,
		},
	}).First(&envelope).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Envelope{}, err
	}

	return envelope, nil
}

// getEnvelopeResources returns all categories for the reuqested budget4.
func getEnvelopeResources(c *gin.Context) ([]models.Envelope, error) {
	var envelopes []models.Envelope

	categories, err := getCategoryResources(c)
	if err != nil {
		return []models.Envelope{}, err
	}

	// Get envelopes for all categories
	for _, category := range categories {
		var e []models.Envelope

		models.DB.Where(&models.Envelope{
			EnvelopeCreate: models.EnvelopeCreate{
				CategoryID: category.ID,
			},
		}).Find(&e)

		envelopes = append(envelopes, e...)
	}

	return envelopes, nil
}

func newEnevlopeResponse(c *gin.Context) EnvelopeResponse {
	budget, _ := getBudgetResource(c)
	category, _ := getCategoryResource(c)
	envelope, _ := getEnvelopeResource(c)

	url := httputil.RequestPathV1(c) + fmt.Sprintf("/budgets/%d/categories/%d/envelopes/%d", budget.ID, category.ID, envelope.ID)

	return EnvelopeResponse{
		Data: envelope,
		Links: EnvelopeLinks{
			Allocations: url + "/allocations",
			Month:       url + "/YYYY-MM",
		},
	}
}
