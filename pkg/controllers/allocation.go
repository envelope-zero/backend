package controllers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/envelope-zero/backend/internal/types"
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
	Month      string          `form:"month"`
	Amount     decimal.Decimal `form:"amount"`
	EnvelopeID string          `form:"envelope"`
}

func (f AllocationQueryFilter) Parse(c *gin.Context) (models.AllocationCreate, bool) {
	envelopeID, ok := httputil.UUIDFromString(c, f.EnvelopeID)
	if !ok {
		return models.AllocationCreate{}, false
	}

	var month QueryMonth
	if err := c.Bind(&month); err != nil {
		httperrors.Handler(c, err)
		return models.AllocationCreate{}, false
	}

	return models.AllocationCreate{
		Month:      types.MonthOf(month.Month),
		Amount:     f.Amount,
		EnvelopeID: envelopeID,
	}, true
}

// RegisterAllocationRoutes registers the routes for allocations with
// the RouterGroup that is passed.
func (co Controller) RegisterAllocationRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsAllocationList)
		r.GET("", co.GetAllocations)
		r.POST("", co.CreateAllocation)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:allocationId", co.OptionsAllocationDetail)
		r.GET("/:allocationId", co.GetAllocation)
		r.PATCH("/:allocationId", co.UpdateAllocation)
		r.DELETE("/:allocationId", co.DeleteAllocation)
	}
}

// OptionsAllocationList returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Allocations
//	@Success		204
//	@Router			/v1/allocations [options]
func (co Controller) OptionsAllocationList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsAllocationDetail returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Allocations
//	@Success		204
//	@Param			allocationId	path	string	true	"ID formatted as string"
//	@Router			/v1/allocations/{allocationId} [options]
func (co Controller) OptionsAllocationDetail(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := co.getAllocationObject(c, p)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// CreateAllocation creates a new allocation
//
//	@Summary		Create allocations
//	@Description	Create a new allocation of funds to an envelope for a specific month
//	@Tags			Allocations
//	@Produce		json
//	@Success		201	{object}	AllocationResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			allocation	body		models.AllocationCreate	true	"Allocation"
//	@Router			/v1/allocations [post]
func (co Controller) CreateAllocation(c *gin.Context) {
	var allocation models.Allocation

	err := httputil.BindData(c, &allocation)
	if err != nil {
		return
	}

	_, ok := co.getEnvelopeResource(c, allocation.EnvelopeID)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Create(&allocation)) {
		return
	}

	allocationObject, _ := co.getAllocationObject(c, allocation.ID)
	c.JSON(http.StatusCreated, AllocationResponse{Data: allocationObject})
}

// GetAllocations returns a list of allocations matching the search parameters
//
//	@Summary		Get allocations
//	@Description	Returns a list of allocations
//	@Tags			Allocations
//	@Produce		json
//	@Success		200	{object}	AllocationListResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/allocations [get]
//	@Param			month		query	string	false	"Filter by month"
//	@Param			amount		query	string	false	"Filter by amount"
//	@Param			envelope	query	string	false	"Filter by envelope ID"
func (co Controller) GetAllocations(c *gin.Context) {
	var filter AllocationQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the parameters set in the query string
	queryFields, _ := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, ok := filter.Parse(c)
	if !ok {
		return
	}

	var allocations []models.Allocation
	if !queryWithRetry(c, co.DB.Where(&models.Allocation{
		AllocationCreate: create,
	}, queryFields...).Find(&allocations)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	allocationObjects := make([]Allocation, 0)

	for _, allocation := range allocations {
		o, _ := co.getAllocationObject(c, allocation.ID)
		allocationObjects = append(allocationObjects, o)
	}

	c.JSON(http.StatusOK, AllocationListResponse{Data: allocationObjects})
}

// GetAllocation returns data about a specific allocation
//
//	@Summary		Get allocation
//	@Description	Returns a specific allocation
//	@Tags			Allocations
//	@Produce		json
//	@Success		200	{object}	AllocationResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			allocationId	path		string	true	"ID formatted as string"
//	@Router			/v1/allocations/{allocationId} [get]
func (co Controller) GetAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocationObject, ok := co.getAllocationObject(c, p)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, AllocationResponse{Data: allocationObject})
}

// UpdateAllocation updates allocation data
//
//	@Summary		Update allocation
//	@Description	Update an allocation. Only values to be updated need to be specified.
//	@Tags			Allocations
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	AllocationResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			allocationId	path		string					true	"ID formatted as string"
//	@Param			allocation		body		models.AllocationCreate	true	"Allocation"
//	@Router			/v1/allocations/{allocationId} [patch]
func (co Controller) UpdateAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocation, ok := co.getAllocationResource(c, p)
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

	if !queryWithRetry(c, co.DB.Model(&allocation).Select("", updateFields...).Updates(data)) {
		return
	}

	allocationObject, _ := co.getAllocationObject(c, allocation.ID)
	c.JSON(http.StatusOK, AllocationResponse{Data: allocationObject})
}

// DeleteAllocation deletes an allocation
//
//	@Summary		Delete allocation
//	@Description	Deletes an allocation
//	@Tags			Allocations
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			allocationId	path		string	true	"ID formatted as string"
//	@Router			/v1/allocations/{allocationId} [delete]
func (co Controller) DeleteAllocation(c *gin.Context) {
	p, err := uuid.Parse(c.Param("allocationId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocation, ok := co.getAllocationResource(c, p)
	if !ok {
		return
	}

	// Allocations are hard deleted instantly to avoid conflicts for the UNIQUE(id,month)
	if !queryWithRetry(c, co.DB.Unscoped().Delete(&allocation)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// getAllocationResource verifies that the request URI is valid for the transaction and returns it.
func (co Controller) getAllocationResource(c *gin.Context, id uuid.UUID) (models.Allocation, bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "no allocation ID specified")
		return models.Allocation{}, false
	}

	var allocation models.Allocation

	if !queryWithRetry(c, co.DB.First(&allocation, &models.Allocation{
		DefaultModel: models.DefaultModel{
			ID: id,
		},
	}), "No allocation found for the specified ID") {
		return models.Allocation{}, false
	}

	return allocation, true
}

func (co Controller) getAllocationObject(c *gin.Context, id uuid.UUID) (Allocation, bool) {
	resource, ok := co.getAllocationResource(c, id)
	if !ok {
		return Allocation{}, false
	}

	return Allocation{
		resource,
		AllocationLinks{
			Self: fmt.Sprintf("%s/v1/allocations/%s", c.GetString("baseURL"), id),
		},
	}, true
}
