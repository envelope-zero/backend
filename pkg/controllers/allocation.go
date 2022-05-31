package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/requestid"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
)

type AllocationResponse struct {
	Data Allocation `json:"data"`
}

type AllocationListResponse struct {
	Data []Allocation `json:"data"`
}

type Allocation struct {
	models.Allocation
	Links AllocationLinks `json:"links"`
}

type AllocationLinks struct {
	Self string `json:"self" example:"https://example.com/api/v1/allocations/902cd93c-3724-4e46-8540-d014131282fc"`
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
// @Router       /v1/allocations [options]
func OptionsAllocationList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Allocations
// @Success      204
// @Param        allocationId  path  string  true  "ID formatted as string"
// @Router       /v1/allocations/{allocationId} [options]
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
// @Param        allocation    body      models.AllocationCreate  true  "Allocation"
// @Router       /v1/allocations [post]
func CreateAllocation(c *gin.Context) {
	var allocation models.Allocation

	err := httputil.BindData(c, &allocation)
	if err != nil {
		return
	}

	_, err = getEnvelopeResource(c, allocation.EnvelopeID)
	if err != nil {
		return
	}

	result := database.DB.Create(&allocation)

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

	allocationObject, _ := getAllocationObject(c, allocation.ID)
	c.JSON(http.StatusCreated, AllocationResponse{Data: allocationObject})
}

// @Summary      Get all allocations for an envelope
// @Description  Returns all allocations for an envelope
// @Tags         Allocations
// @Produce      json
// @Success      200  {object}  AllocationListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/allocations [get]
func GetAllocations(c *gin.Context) {
	var allocations []models.Allocation

	database.DB.Find(&allocations)

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	allocationObjects := make([]Allocation, 0)

	for _, allocation := range allocations {
		o, _ := getAllocationObject(c, allocation.ID)
		allocationObjects = append(allocationObjects, o)
	}

	c.JSON(http.StatusOK, AllocationListResponse{Data: allocationObjects})
}

// @Summary      Get allocation
// @Description  Returns an allocation by its ID
// @Tags         Allocations
// @Produce      json
// @Success      200  {object}  AllocationResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500           {object}  httputil.HTTPError
// @Param        allocationId  path      string  true  "ID formatted as string"
// @Router       /v1/allocations/{allocationId} [get]
func GetAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	allocationObject, err := getAllocationObject(c, p)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, AllocationResponse{Data: allocationObject})
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
// @Param        allocationId  path      string                   true  "ID formatted as string"
// @Param        allocation  body      models.AllocationCreate  true  "Allocation"
// @Router       /v1/allocations/{allocationId} [patch]
func UpdateAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	allocation, err := getAllocationResource(c, p)
	if err != nil {
		return
	}

	var data models.Allocation
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	database.DB.Model(&allocation).Updates(data)
	allocationObject, _ := getAllocationObject(c, allocation.ID)

	c.JSON(http.StatusOK, AllocationResponse{Data: allocationObject})
}

// @Summary      Delete an allocation
// @Description  Deletes an existing allocation
// @Tags         Allocations
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        allocationId  path      string  true  "ID formatted as string"
// @Router       /v1/allocations/{allocationId} [delete]
func DeleteAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	allocation, err := getAllocationResource(c, p)
	if err != nil {
		return
	}

	database.DB.Delete(&allocation)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getAllocationResource verifies that the request URI is valid for the transaction and returns it.
func getAllocationResource(c *gin.Context, id uuid.UUID) (models.Allocation, error) {
	var allocation models.Allocation

	err := database.DB.First(&allocation, &models.Allocation{
		Model: models.Model{
			ID: id,
		},
	}).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Allocation{}, err
	}

	return allocation, nil
}

func getAllocationObject(c *gin.Context, id uuid.UUID) (Allocation, error) {
	resource, err := getAllocationResource(c, id)
	if err != nil {
		return Allocation{}, err
	}

	return Allocation{
		resource,
		getAllocationLinks(c, id),
	}, nil
}

// getAllocationLinks returns a BudgetLinks struct.
//
// This function is only needed for getAllocationObject as we cannot create an instance of Allocation
// with mixed named and unnamed parameters.
func getAllocationLinks(c *gin.Context, id uuid.UUID) AllocationLinks {
	url := httputil.RequestPathV1(c) + fmt.Sprintf("/allocations/%s", id)

	return AllocationLinks{
		Self: url,
	}
}
