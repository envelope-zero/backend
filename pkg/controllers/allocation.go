package controllers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
)

type Allocation struct {
	models.Allocation
	Links struct {
		Self string `json:"self" example:"https://example.com/api/v1/allocations/902cd93c-3724-4e46-8540-d014131282fc"` // The allocation itself
	} `json:"links" gorm:"-"`
}

func (a *Allocation) links(c *gin.Context) {
	a.Links.Self = fmt.Sprintf("%s/v1/allocations/%s", c.GetString(string(database.ContextURL)), a.ID)
}

func (co Controller) getAllocation(c *gin.Context, id uuid.UUID) (Allocation, bool) {
	m, ok := getResourceByIDAndHandleErrors[models.Allocation](c, co, id)
	if !ok {
		return Allocation{}, false
	}

	a := Allocation{
		Allocation: m,
	}

	a.links(c)
	return a, true
}

type AllocationResponse struct {
	Data Allocation `json:"data"` // List of allocations
}

type AllocationListResponse struct {
	Data []Allocation `json:"data"` // Data for the allocation
}

type AllocationQueryFilter struct {
	Month      string          `form:"month"`    // By month
	Amount     decimal.Decimal `form:"amount"`   // By exact amount
	EnvelopeID string          `form:"envelope"` // By the Envelope ID
}

func (f AllocationQueryFilter) Parse(c *gin.Context) (models.AllocationCreate, bool) {
	envelopeID, ok := httputil.UUIDFromStringHandleErrors(c, f.EnvelopeID)
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
		r.OPTIONS("/:id", co.OptionsAllocationDetail)
		r.GET("/:id", co.GetAllocation)
		r.PATCH("/:id", co.UpdateAllocation)
		r.DELETE("/:id", co.DeleteAllocation)
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
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v1/allocations/{id} [options]
func (co Controller) OptionsAllocationDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.Allocation](c, co, id)
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
//	@Success		201			{object}	AllocationResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			allocation	body		models.AllocationCreate	true	"Allocation"
//	@Router			/v1/allocations [post]
func (co Controller) CreateAllocation(c *gin.Context) {
	var create models.AllocationCreate

	err := httputil.BindDataHandleErrors(c, &create)
	if err != nil {
		return
	}

	a := models.Allocation{
		AllocationCreate: create,
	}

	_, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, a.EnvelopeID)
	if !ok {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Create(&a)) {
		return
	}

	o, ok := co.getAllocation(c, a.ID)
	if !ok {
		return
	}

	c.JSON(http.StatusCreated, AllocationResponse{Data: o})
}

// GetAllocations returns a list of allocations matching the search parameters
//
//	@Summary		Get allocations
//	@Description	Returns a list of allocations
//	@Tags			Allocations
//	@Produce		json
//	@Success		200	{object}	AllocationListResponse
//	@Failure		400	{object}	httperrors.HTTPError
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
	if !queryAndHandleErrors(c, co.DB.Where(&models.Allocation{
		AllocationCreate: create,
	}, queryFields...).Find(&allocations)) {
		return
	}

	s := make([]Allocation, 0)
	for _, allocation := range allocations {
		a, ok := co.getAllocation(c, allocation.ID)
		if !ok {
			return
		}

		s = append(s, a)
	}

	c.JSON(http.StatusOK, AllocationListResponse{Data: s})
}

// GetAllocation returns data about a specific allocation
//
//	@Summary		Get allocation
//	@Description	Returns a specific allocation
//	@Tags			Allocations
//	@Produce		json
//	@Success		200	{object}	AllocationResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v1/allocations/{id} [get]
func (co Controller) GetAllocation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocation, ok := getResourceByIDAndHandleErrors[models.Allocation](c, co, id)
	if !ok {
		return
	}

	a, ok := co.getAllocation(c, allocation.ID)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, AllocationResponse{Data: a})
}

// UpdateAllocation updates allocation data
//
//	@Summary		Update allocation
//	@Description	Update an allocation. Only values to be updated need to be specified.
//	@Tags			Allocations
//	@Accept			json
//	@Produce		json
//	@Success		200			{object}	AllocationResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			id			path		string					true	"ID formatted as string"
//	@Param			allocation	body		models.AllocationCreate	true	"Allocation"
//	@Router			/v1/allocations/{id} [patch]
func (co Controller) UpdateAllocation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocation, ok := getResourceByIDAndHandleErrors[models.Allocation](c, co, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFieldsHandleErrors(c, models.AllocationCreate{})
	if err != nil {
		return
	}

	var data models.Allocation
	if err := httputil.BindDataHandleErrors(c, &data.AllocationCreate); err != nil {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Model(&allocation).Select("", updateFields...).Updates(data)) {
		return
	}

	a, ok := co.getAllocation(c, allocation.ID)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, AllocationResponse{Data: a})
}

// DeleteAllocation deletes an allocation
//
//	@Summary		Delete allocation
//	@Description	Deletes an allocation
//	@Tags			Allocations
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v1/allocations/{id} [delete]
func (co Controller) DeleteAllocation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	allocation, ok := getResourceByIDAndHandleErrors[models.Allocation](c, co, id)
	if !ok {
		return
	}

	// Allocations are hard deleted instantly to avoid conflicts for the UNIQUE(id,month)
	if !queryAndHandleErrors(c, co.DB.Unscoped().Delete(&allocation)) {
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
