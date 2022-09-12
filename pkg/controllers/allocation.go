package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/envelope-zero/backend/pkg/database"
	"github.com/envelope-zero/backend/pkg/httperrors"
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

type AllocationQueryFilter struct {
	Month      time.Time       `form:"month"`
	Amount     decimal.Decimal `form:"amount"`
	EnvelopeID string          `form:"envelope"`
}

func (f AllocationQueryFilter) ToCreate(c *gin.Context) (models.AllocationCreate, error) {
	envelopeID, err := httputil.UUIDFromString(c, f.EnvelopeID)
	if err != nil {
		return models.AllocationCreate{}, err
	}

	return models.AllocationCreate{
		Month:      f.Month,
		Amount:     f.Amount,
		EnvelopeID: envelopeID,
	}, nil
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

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Allocations
// @Success     204
// @Router      /v1/allocations [options]
func OptionsAllocationList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Allocations
// @Success     204
// @Param       allocationId path string true "ID formatted as string"
// @Router      /v1/allocations/{allocationId} [options]
func OptionsAllocationDetail(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := getAllocationObject(c, p)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// @Summary     Create allocations
// @Description Create a new allocation of funds to an envelope for a specific month
// @Tags        Allocations
// @Produce     json
// @Success     201 {object} AllocationResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500        {object} httperrors.HTTPError
// @Param       allocation body     models.AllocationCreate true "Allocation"
// @Router      /v1/allocations [post]
func CreateAllocation(c *gin.Context) {
	var allocation models.Allocation

	err := httputil.BindData(c, &allocation)
	if err != nil {
		return
	}

	// Ignore every field that is not Year or Month
	allocation.Month = time.Date(allocation.Month.Year(), allocation.Month.Month(), 1, 0, 0, 0, 0, time.UTC)

	_, err = getEnvelopeResource(c, allocation.EnvelopeID)
	if err != nil {
		return
	}

	if !queryWithRetry(c, database.DB.Create(&allocation)) {
		return
	}

	allocationObject, _ := getAllocationObject(c, allocation.ID)
	c.JSON(http.StatusCreated, AllocationResponse{Data: allocationObject})
}

// @Summary     Get allocations
// @Description Returns a list of allocations
// @Tags        Allocations
// @Produce     json
// @Success     200 {object} AllocationListResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500 {object} httperrors.HTTPError
// @Router      /v1/allocations [get]
// @Param       month    query string false "Filter by month"
// @Param       amount   query string false "Filter by amount"
// @Param       envelope query string false "Filter by envelope ID"
func GetAllocations(c *gin.Context) {
	var filter AllocationQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the parameters set in the query string
	queryFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate(c)
	if err != nil {
		return
	}

	var allocations []models.Allocation
	if !queryWithRetry(c, database.DB.Where(&models.Allocation{
		AllocationCreate: create,
	}, queryFields...).Find(&allocations)) {
		return
	}

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

// @Summary     Get allocation
// @Description Returns a specific allocation
// @Tags        Allocations
// @Produce     json
// @Success     200 {object} AllocationResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500          {object} httperrors.HTTPError
// @Param       allocationId path     string true "ID formatted as string"
// @Router      /v1/allocations/{allocationId} [get]
func GetAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocationObject, ok := getAllocationObject(c, p)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, AllocationResponse{Data: allocationObject})
}

// @Summary     Update allocation
// @Description Update an allocation. Only values to be updated need to be specified.
// @Tags        Allocations
// @Accept      json
// @Produce     json
// @Success     200 {object} AllocationResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500          {object} httperrors.HTTPError
// @Param       allocationId path     string                  true "ID formatted as string"
// @Param       allocation   body     models.AllocationCreate true "Allocation"
// @Router      /v1/allocations/{allocationId} [patch]
func UpdateAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocation, ok := getAllocationResource(c, p)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.AllocationCreate{})
	if err != nil {
		return
	}

	var data models.Allocation
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	// Ignore every field that is not Year or Month
	allocation.Month = time.Date(allocation.Month.Year(), allocation.Month.Month(), 1, 0, 0, 0, 0, time.UTC)

	if !queryWithRetry(c, database.DB.Model(&allocation).Select("", updateFields...).Updates(data)) {
		return
	}

	allocationObject, _ := getAllocationObject(c, allocation.ID)
	c.JSON(http.StatusOK, AllocationResponse{Data: allocationObject})
}

// @Summary     Delete allocation
// @Description Deletes an allocation
// @Tags        Allocations
// @Success     204
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500          {object} httperrors.HTTPError
// @Param       allocationId path     string true "ID formatted as string"
// @Router      /v1/allocations/{allocationId} [delete]
func DeleteAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocation, ok := getAllocationResource(c, p)
	if !ok {
		return
	}

	// Allocations are hard deleted instantly to avoid conflicts for the UNIQUE(id,month)
	if !queryWithRetry(c, database.DB.Unscoped().Delete(&allocation)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// getAllocationResource verifies that the request URI is valid for the transaction and returns it.
func getAllocationResource(c *gin.Context, id uuid.UUID) (models.Allocation, bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "no allocation ID specified")
		return models.Allocation{}, false
	}

	var allocation models.Allocation

	if !queryWithRetry(c, database.DB.First(&allocation, &models.Allocation{
		Model: models.Model{
			ID: id,
		},
	}), "No allocation found for the specified ID") {
		return models.Allocation{}, false
	}

	return allocation, true
}

func getAllocationObject(c *gin.Context, id uuid.UUID) (Allocation, bool) {
	resource, ok := getAllocationResource(c, id)
	if !ok {
		return Allocation{}, false
	}

	return Allocation{
		resource,
		getAllocationLinks(c, id),
	}, true
}

// getAllocationLinks returns a BudgetLinks struct.
//
// This function is only needed for getAllocationObject as we cannot create an instance of Allocation
// with mixed named and unnamed parameters.
func getAllocationLinks(c *gin.Context, id uuid.UUID) AllocationLinks {
	url := fmt.Sprintf("%s/v1/allocations/%s", c.GetString("baseURL"), id)

	return AllocationLinks{
		Self: url,
	}
}
