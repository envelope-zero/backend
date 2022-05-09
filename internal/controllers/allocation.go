package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-contrib/requestid"
	"github.com/rs/zerolog/log"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type AllocationResponse struct {
	Data models.Allocation `json:"data"`
}

type AllocationListResponse struct {
	Data []models.Allocation `json:"data"`
}

// RegisterAllocationRoutes registers the routes for allocations with
// the RouterGroup that is passed.
func RegisterAllocationRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsAllocationList)
		r.GET("", GetAllocations)
		r.POST("", CreateAllocation)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:allocationId", OptionsAllocationDetail)
		r.GET("/:allocationId", GetAllocation)
		r.PATCH("/:allocationId", UpdateAllocation)
		r.DELETE("/:allocationId", DeleteAllocation)
	}
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Allocations
// @Success      204
// @Param        budgetId    path  uint64  true  "ID of the budget"
// @Param        categoryId    path  uint64  true  "ID of the category"
// @Param        envelopeId    path  uint64  true  "ID of the envelope"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId}/allocations [options]
func OptionsAllocationList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Allocations
// @Success      204
// @Param        budgetId      path  uint64  true  "ID of the budget"
// @Param        categoryId  path  uint64  true  "ID of the category"
// @Param        envelopeId  path  uint64  true  "ID of the envelope"
// @Param        allocationId  path  uint64  true  "ID of the allocation"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId}/allocations/{allocationId} [options]
func OptionsAllocationDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// @Summary      Create allocations
// @Description  Create a new allocation of funds to an envelope for a specific month
// @Tags         Allocations
// @Produce      json
// @Success      201  {object}  AllocationResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500           {object}  httputil.HTTPError
// @Param        budgetId      path      uint64                   true  "ID of the budget"
// @Param        categoryId    path      uint64                   true  "ID of the category"
// @Param        envelopeId    path      uint64                   true  "ID of the envelope"
// @Param        allocation    body      models.AllocationCreate  true  "Allocation"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId}/allocations [post]
func CreateAllocation(c *gin.Context) {
	var data models.Allocation

	err := httputil.BindData(c, &data)
	if err != nil {
		return
	}

	data.EnvelopeID, err = httputil.ParseID(c, "envelopeId")
	if err != nil {
		return
	}
	result := models.DB.Create(&data)

	if result.Error != nil {
		// By default, we assume a server error
		errMessage := "There was an error processing your request, please contact your server administrator"
		status := http.StatusInternalServerError

		// Set helpful error messages for known errors
		if strings.Contains(result.Error.Error(), "UNIQUE constraint failed: allocations.month, allocations.year") {
			errMessage = "You can not create multiple allocations for the same month"
			status = http.StatusBadRequest
		} else if strings.Contains(result.Error.Error(), "CHECK constraint failed: month_valid") {
			errMessage = "The month must be between 1 and 12"
			status = http.StatusBadRequest
		}

		// Print the error to the server log if it’s a server error
		if status == http.StatusInternalServerError {
			log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", result.Error, result.Error.Error())
		}

		httputil.NewError(c, status, errors.New(errMessage))
		return
	}

	c.JSON(http.StatusCreated, AllocationResponse{Data: data})
}

// @Summary      Get all allocations for an envelope
// @Description  Returns all allocations for an envelope
// @Tags         Allocations
// @Produce      json
// @Success      200  {object}  AllocationListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId    path      uint64  true  "ID of the budget"
// @Param        categoryId  path      uint64  true  "ID of the category"
// @Param        envelopeId  path      uint64  true  "ID of the envelope"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId}/allocations [get]
func GetAllocations(c *gin.Context) {
	var allocations []models.Allocation

	// Check if the envelope exists
	envelope, err := getEnvelopeResource(c)
	if err != nil {
		return
	}

	models.DB.Where(&models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
		},
	}).Find(&allocations)

	c.JSON(http.StatusOK, AllocationListResponse{Data: allocations})
}

// @Summary      Get allocation
// @Description  Returns an allocation by its ID
// @Tags         Allocations
// @Produce      json
// @Success      200  {object}  AllocationResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500           {object}  httputil.HTTPError
// @Param        budgetId      path      uint64  true  "ID of the budget"
// @Param        categoryId    path      uint64  true  "ID of the category"
// @Param        envelopeId  path      uint64                   true  "ID of the envelope"
// @Param        allocationId  path      uint64  true  "ID of the allocation"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId}/allocations/{allocationId} [get]
func GetAllocation(c *gin.Context) {
	allocation, err := getAllocationResource(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, AllocationResponse{Data: allocation})
}

// @Summary      Update an allocation
// @Description  Update an existing allocation. Only values to be updated need to be specified.
// @Tags         Allocations
// @Accept       json
// @Produce      json
// @Success      200  {object}  AllocationResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500           {object}  httputil.HTTPError
// @Param        budgetId    path      uint64                   true  "ID of the budget"
// @Param        categoryId  path      uint64                   true  "ID of the category"
// @Param        envelopeId    path      uint64  true  "ID of the envelope"
// @Param        allocationId  path      uint64                   true  "ID of the allocation"
// @Param        allocation  body      models.AllocationCreate  true  "Allocation"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId}/allocations/{allocationId} [patch]
func UpdateAllocation(c *gin.Context) {
	allocation, err := getAllocationResource(c)
	if err != nil {
		return
	}

	var data models.Allocation
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	models.DB.Model(&allocation).Updates(data)
	c.JSON(http.StatusOK, AllocationResponse{Data: allocation})
}

// @Summary      Delete an allocation
// @Description  Deletes an existing allocation
// @Tags         Allocations
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId      path      uint64  true  "ID of the budget"
// @Param        categoryId    path      uint64  true  "ID of the category"
// @Param        envelopeId    path      uint64  true  "ID of the envelope"
// @Param        allocationId  path      uint64  true  "ID of the allocation"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId}/envelopes/{envelopeId}/allocations/{allocationId} [delete]
func DeleteAllocation(c *gin.Context) {
	allocation, err := getAllocationResource(c)
	if err != nil {
		return
	}

	models.DB.Delete(&allocation)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getAllocationResource verifies that the request URI is valid for the transaction and returns it.
func getAllocationResource(c *gin.Context) (models.Allocation, error) {
	var allocation models.Allocation

	envelope, err := getEnvelopeResource(c)
	if err != nil {
		return models.Allocation{}, err
	}

	allocationID, err := httputil.ParseID(c, "allocationId")
	if err != nil {
		return models.Allocation{}, err
	}

	err = models.DB.First(&allocation, &models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
		},
		Model: models.Model{
			ID: allocationID,
		},
	}).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Allocation{}, err
	}

	return allocation, nil
}
