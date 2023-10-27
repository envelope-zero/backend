package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Category struct {
	models.Category
	Envelopes []Envelope `json:"envelopes" gorm:"-"` // Envelopes for the category
	Links     struct {
		Self      string `json:"self" example:"https://example.com/api/v1/categories/3b1ea324-d438-4419-882a-2fc91d71772f"`              // The category itself
		Envelopes string `json:"envelopes" example:"https://example.com/api/v1/envelopes?category=3b1ea324-d438-4419-882a-2fc91d71772f"` // Envelopes for this category
	} `json:"links" gorm:"-"`
}

func (c *Category) links(context *gin.Context) {
	url := context.GetString(string(database.ContextURL))

	c.Links.Self = fmt.Sprintf("%s/v1/categories/%s", url, c.ID)
	c.Links.Envelopes = fmt.Sprintf("%s/v1/envelopes?category=%s", url, c.ID)
}

func (co Controller) getCategory(c *gin.Context, id uuid.UUID) (Category, bool) {
	m, ok := getResourceByIDAndHandleErrors[models.Category](c, co, id)
	if !ok {
		return Category{}, false
	}

	cat := Category{
		Category: m,
	}

	eModels, err := m.Envelopes(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return Category{}, false
	}

	envelopes := make([]Envelope, 0)
	for _, e := range eModels {
		o, ok := co.getEnvelope(c, e.ID)
		if !ok {
			return Category{}, false
		}
		envelopes = append(envelopes, o)
	}

	cat.Envelopes = envelopes
	cat.links(c)

	return cat, true
}

type CategoryListResponse struct {
	Data []Category `json:"data"` // List of categories
}

type CategoryResponse struct {
	Data Category `json:"data"` // Data for the category
}

type CategoryQueryFilter struct {
	Name     string `form:"name" filterField:"false"`   // By name
	BudgetID string `form:"budget"`                     // By ID of the budget
	Note     string `form:"note" filterField:"false"`   // By note
	Hidden   bool   `form:"hidden"`                     // Is the category archived?
	Search   string `form:"search" filterField:"false"` // By string in name or note
}

func (f CategoryQueryFilter) ToCreate(c *gin.Context) (models.CategoryCreate, bool) {
	budgetID, ok := httputil.UUIDFromString(c, f.BudgetID)
	if !ok {
		return models.CategoryCreate{}, false
	}

	return models.CategoryCreate{
		BudgetID: budgetID,
		Hidden:   f.Hidden,
	}, true
}

// RegisterCategoryRoutes registers the routes for categories with
// the RouterGroup that is passed.
func (co Controller) RegisterCategoryRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsCategoryList)
		r.GET("", co.GetCategories)
		r.POST("", co.CreateCategory)
	}

	// Category with ID
	{
		r.OPTIONS("/:categoryId", co.OptionsCategoryDetail)
		r.GET("/:categoryId", co.GetCategory)
		r.PATCH("/:categoryId", co.UpdateCategory)
		r.DELETE("/:categoryId", co.DeleteCategory)
	}
}

// OptionsCategoryList returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Categories
//	@Success		204
//	@Router			/v1/categories [options]
func (co Controller) OptionsCategoryList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsCategoryDetail returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Categories
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			categoryId	path		string	true	"ID formatted as string"
//	@Router			/v1/categories/{categoryId} [options]
func (co Controller) OptionsCategoryDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := co.getCategory(c, id)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// CreateCategory creates a new category
//
//	@Summary		Create category
//	@Description	Creates a new category
//	@Tags			Categories
//	@Produce		json
//	@Success		201			{object}	CategoryResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			category	body		models.CategoryCreate	true	"Category"
//	@Router			/v1/categories [post]
func (co Controller) CreateCategory(c *gin.Context) {
	var create models.CategoryCreate

	err := httputil.BindData(c, &create)
	if err != nil {
		return
	}

	cat := models.Category{
		CategoryCreate: create,
	}

	_, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, cat.BudgetID)
	if !ok {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Create(&cat)) {
		return
	}

	r, ok := co.getCategory(c, cat.ID)
	if !ok {
		return
	}
	c.JSON(http.StatusCreated, CategoryResponse{Data: r})
}

// GetCategories returns a list of categories filtered by the query parameters
//
//	@Summary		Get categories
//	@Description	Returns a list of categories
//	@Tags			Categories
//	@Produce		json
//	@Success		200	{object}	CategoryListResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/categories [get]
//	@Param			name	query	string	false	"Filter by name"
//	@Param			note	query	string	false	"Filter by note"
//	@Param			budget	query	string	false	"Filter by budget ID"
//	@Param			hidden	query	bool	false	"Is the category hidden?"
//	@Param			search	query	string	false	"Search for this text in name and note"
func (co Controller) GetCategories(c *gin.Context) {
	var filter CategoryQueryFilter

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we are filtering for
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, ok := filter.ToCreate(c)
	if !ok {
		return
	}

	query := co.DB.Where(&models.Category{
		CategoryCreate: create,
	}, queryFields...)

	query = stringFilters(co.DB, query, setFields, filter.Name, filter.Note, filter.Search)

	var categories []models.Category
	if !queryAndHandleErrors(c, query.Find(&categories)) {
		return
	}

	r := make([]Category, 0)
	for _, category := range categories {
		o, ok := co.getCategory(c, category.ID)
		if !ok {
			return
		}
		r = append(r, o)
	}

	c.JSON(http.StatusOK, CategoryListResponse{Data: r})
}

// GetCategory returns data for a specific category
//
//	@Summary		Get category
//	@Description	Returns a specific category
//	@Tags			Categories
//	@Produce		json
//	@Success		200			{object}	CategoryResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			categoryId	path		string	true	"ID formatted as string"
//	@Router			/v1/categories/{categoryId} [get]
func (co Controller) GetCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	r, ok := co.getCategory(c, id)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, CategoryResponse{Data: r})
}

// UpdateCategory updates data for a specific category
//
//	@Summary		Update category
//	@Description	Update an existing category. Only values to be updated need to be specified.
//	@Tags			Categories
//	@Accept			json
//	@Produce		json
//	@Success		200			{object}	CategoryResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			categoryId	path		string					true	"ID formatted as string"
//	@Param			category	body		models.CategoryCreate	true	"Category"
//	@Router			/v1/categories/{categoryId} [patch]
func (co Controller) UpdateCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	category, ok := getResourceByIDAndHandleErrors[models.Category](c, co, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.CategoryCreate{})
	if err != nil {
		return
	}

	var data models.Category
	if err := httputil.BindData(c, &data.CategoryCreate); err != nil {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Model(&category).Select("", updateFields...).Updates(data)) {
		return
	}

	r, ok := co.getCategory(c, category.ID)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, CategoryResponse{Data: r})
}

// DeleteCategory deletes a specific category
//
//	@Summary		Delete category
//	@Description	Deletes a category
//	@Tags			Categories
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			categoryId	path		string	true	"ID formatted as string"
//	@Router			/v1/categories/{categoryId} [delete]
func (co Controller) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	category, ok := getResourceByIDAndHandleErrors[models.Category](c, co, id)
	if !ok {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Delete(&category)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
