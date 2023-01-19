package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/envelope-zero/backend/v2/pkg/httputil"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CategoryListResponse struct {
	Data []Category `json:"data"`
}

type CategoryResponse struct {
	Data Category `json:"data"`
}

type Category struct {
	models.Category
	Envelopes []models.Envelope `json:"envelopes"`
}

type CategoryQueryFilter struct {
	Name     string `form:"name" filterField:"false"`
	BudgetID string `form:"budget"`
	Note     string `form:"note" filterField:"false"`
	Hidden   bool   `form:"hidden"`
	Search   string `form:"search" filterField:"false"`
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
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
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
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Param			categoryId	path	string	true	"ID formatted as string"
//	@Router			/v1/categories/{categoryId} [options]
func (co Controller) OptionsCategoryDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := co.getCategoryObject(c, id)
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
//	@Success		201	{object}	CategoryResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			category	body		models.CategoryCreate	true	"Category"
//	@Router			/v1/categories [post]
func (co Controller) CreateCategory(c *gin.Context) {
	var category models.Category

	err := httputil.BindData(c, &category)
	if err != nil {
		return
	}

	_, ok := co.getBudgetResource(c, category.BudgetID)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Create(&category)) {
		return
	}

	categoryObject, _ := co.getCategoryObject(c, category.ID)
	c.JSON(http.StatusCreated, CategoryResponse{Data: categoryObject})
}

// GetCategories returns a list of categories filtered by the query parameters
//
//	@Summary		Get categories
//	@Description	Returns a list of categories
//	@Tags			Categories
//	@Produce		json
//	@Success		200	{object}	CategoryListResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
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
	if !queryWithRetry(c, query.Find(&categories)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	categoryObjects := make([]Category, 0)

	for _, category := range categories {
		o, _ := co.getCategoryObject(c, category.ID)
		categoryObjects = append(categoryObjects, o)
	}

	c.JSON(http.StatusOK, CategoryListResponse{Data: categoryObjects})
}

// GetCategory returns data for a specific category
//
//	@Summary		Get category
//	@Description	Returns a specific category
//	@Tags			Categories
//	@Produce		json
//	@Success		200	{object}	CategoryResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			categoryId	path		string	true	"ID formatted as string"
//	@Router			/v1/categories/{categoryId} [get]
func (co Controller) GetCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	categoryObject, ok := co.getCategoryObject(c, id)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, CategoryResponse{Data: categoryObject})
}

// UpdateCategory updates data for a specific category
//
//	@Summary		Update category
//	@Description	Update an existing category. Only values to be updated need to be specified.
//	@Tags			Categories
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	CategoryResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
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

	category, ok := co.getCategoryResource(c, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.CategoryCreate{})
	if err != nil {
		return
	}

	var data models.Category
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	if !queryWithRetry(c, co.DB.Model(&category).Select("", updateFields...).Updates(data)) {
		return
	}

	categoryObject, _ := co.getCategoryObject(c, category.ID)
	c.JSON(http.StatusOK, CategoryResponse{Data: categoryObject})
}

// DeleteCategory deletes a specific category
//
//	@Summary		Delete category
//	@Description	Deletes a category
//	@Tags			Categories
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			categoryId	path		string	true	"ID formatted as string"
//	@Router			/v1/categories/{categoryId} [delete]
func (co Controller) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	category, ok := co.getCategoryResource(c, id)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&category)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

func (co Controller) getCategoryResource(c *gin.Context, id uuid.UUID) (models.Category, bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "No category ID specified")
		return models.Category{}, false
	}

	var category models.Category

	if !queryWithRetry(c, co.DB.Where(&models.Category{
		DefaultModel: models.DefaultModel{
			ID: id,
		},
	}).First(&category), "No category found for the specified ID") {
		return models.Category{}, false
	}

	return category, true
}

// getCategoryResources returns all categories for the requested budget.
func (co Controller) getCategoryResources(c *gin.Context, id uuid.UUID) ([]models.Category, bool) {
	var categories []models.Category

	if !queryWithRetry(c, co.DB.Where(&models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: id,
		},
	}).Find(&categories)) {
		return []models.Category{}, false
	}

	return categories, true
}

func (co Controller) getCategoryObject(c *gin.Context, id uuid.UUID) (Category, bool) {
	resource, ok := co.getCategoryResource(c, id)
	if !ok {
		return Category{}, false
	}

	var envelopes []models.Envelope
	err := co.DB.Where(&models.Envelope{EnvelopeCreate: models.EnvelopeCreate{CategoryID: id}}).Find(&envelopes).Error
	if err != nil {
		httperrors.Handler(c, err)
		return Category{}, false
	}

	return Category{
		resource,
		envelopes,
	}, true
}
