package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	Self        string `json:"self" example:"https://example.com/api/v1/envelopes/45b6b5b9-f746-4ae9-b77b-7688b91f8166"`
	Allocations string `json:"allocations" example:"https://example.com/api/v1/allocations?envelope=45b6b5b9-f746-4ae9-b77b-7688b91f8166"`
	Month       string `json:"month" example:"https://example.com/api/v1/envelopes/45b6b5b9-f746-4ae9-b77b-7688b91f8166/YYYY-MM"` // This will always end in 'YYYY-MM' for clients to use replace with actual numbers.
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
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Envelopes
// @Success      204
// @Router       /v1/envelopes [options]
func OptionsEnvelopeList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Envelopes
// @Success      204
// @Param        envelopeId  path  string  true  "ID formatted as string"
// @Router       /v1/envelopes/{envelopeId} [options]
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
// @Failure      500       {object}  httputil.HTTPError
// @Param        envelope  body      models.EnvelopeCreate  true  "Envelope"
// @Router       /v1/envelopes [post]
func CreateEnvelope(c *gin.Context) {
	var envelope models.Envelope

	err := httputil.BindData(c, &envelope)
	if err != nil {
		return
	}

	_, err = getCategoryResource(c, envelope.CategoryID)
	if err != nil {
		return
	}

	database.DB.Create(&envelope)

	envelopeObject, _ := getEnvelopeObject(c, envelope.ID)
	c.JSON(http.StatusCreated, EnvelopeResponse{Data: envelopeObject})
}

// @Summary      Get all envelopes for a category
// @Description  Returns the full list of all envelopes for a specific category
// @Tags         Envelopes
// @Produce      json
// @Success      200  {object}  EnvelopeListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/envelopes [get]
func GetEnvelopes(c *gin.Context) {
	var envelopes []models.Envelope

	database.DB.Find(&envelopes)

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	envelopeObjects := make([]Envelope, 0)

	for _, envelope := range envelopes {
		o, _ := getEnvelopeObject(c, envelope.ID)
		envelopeObjects = append(envelopeObjects, o)
	}

	c.JSON(http.StatusOK, EnvelopeListResponse{Data: envelopeObjects})
}

// @Summary      Get envelope
// @Description  Returns an envelope by its ID
// @Tags         Envelopes
// @Produce      json
// @Success      200  {object}  EnvelopeResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        envelopeId  path      string  true  "ID formatted as string"
// @Router       /v1/envelopes/{envelopeId} [get]
func GetEnvelope(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	envelopeObject, err := getEnvelopeObject(c, p)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, EnvelopeResponse{Data: envelopeObject})
}

// @Summary      Get Envelope month data
// @Description  Returns data about an envelope for a for a specific month
// @Tags         Envelopes
// @Produce      json
// @Success      200  {object}  EnvelopeMonthResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        envelopeId  path      string  true  "ID formatted as string"
// @Param        month       path      string  true  "The month in YYYY-MM format"
// @Router       /v1/envelopes/{envelopeId}/{month} [get]
func GetEnvelopeMonth(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		return
	}

	envelope, _ := getEnvelopeResource(c, p)

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
// @Param        envelopeId  path      string                 true  "ID formatted as string"
// @Param        envelope    body      models.EnvelopeCreate  true  "Envelope"
// @Router       /v1/envelopes/{envelopeId} [patch]
func UpdateEnvelope(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	envelope, err := getEnvelopeResource(c, p)
	if err != nil {
		return
	}

	var data models.Envelope
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	database.DB.Model(&envelope).Updates(data)
	envelopeObject, _ := getEnvelopeObject(c, envelope.ID)
	c.JSON(http.StatusOK, EnvelopeResponse{Data: envelopeObject})
}

// @Summary      Delete an envelope
// @Description  Deletes an existing envelope
// @Tags         Envelopes
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        envelopeId  path      string  true  "ID formatted as string"
// @Router       /v1/envelopes/{envelopeId} [delete]
func DeleteEnvelope(c *gin.Context) {
	p, err := uuid.Parse(c.Param("envelopeId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	envelope, err := getEnvelopeResource(c, p)
	if err != nil {
		return
	}

	database.DB.Delete(&envelope)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getEnvelopeResource verifies that the envelope from the URL parameters exists and returns it.
func getEnvelopeResource(c *gin.Context, id uuid.UUID) (models.Envelope, error) {
	if id == uuid.Nil {
		err := errors.New("No envelope ID specified")
		httputil.NewError(c, http.StatusBadRequest, err)
		return models.Envelope{}, err
	}

	var envelope models.Envelope

	err := database.DB.Where(&models.Envelope{
		Model: models.Model{
			ID: id,
		},
	}).First(&envelope).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Envelope{}, err
	}

	return envelope, nil
}

func getEnvelopeObject(c *gin.Context, id uuid.UUID) (Envelope, error) {
	resource, err := getEnvelopeResource(c, id)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		resource,
		getEnvelopeLinks(c, id),
	}, nil
}

func getEnvelopeObjects(c *gin.Context, categoryID uuid.UUID) ([]Envelope, error) {
	var envelopes []models.Envelope

	err := database.DB.Where(&models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			CategoryID: categoryID,
		},
	}).Find(&envelopes).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return []Envelope{}, err
	}

	var envelopeObjects []Envelope
	for _, envelope := range envelopes {
		o, _ := getEnvelopeObject(c, envelope.ID)
		envelopeObjects = append(envelopeObjects, o)
	}

	return envelopeObjects, nil
}

// getEnvelopeLinks returns a BudgetLinks struct.
//
// This function is only needed for getEnvelopeObject as we cannot create an instance of Envelope
// with mixed named and unnamed parameters.
func getEnvelopeLinks(c *gin.Context, id uuid.UUID) EnvelopeLinks {
	url := httputil.RequestPathV1(c) + fmt.Sprintf("/envelopes/%s", id)

	return EnvelopeLinks{
		Self:        url,
		Allocations: url + "/allocations",
		Month:       url + "/YYYY-MM",
	}
}
