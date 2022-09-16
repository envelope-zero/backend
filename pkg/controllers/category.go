package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
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
	Links     CategoryLinks `json:"links"`
	Envelopes []Envelope    `json:"envelopes"`
}

type CategoryLinks struct {
	Self      string `json:"self" example:"https://example.com/api/v1/categories/3b1ea324-d438-4419-882a-2fc91d71772f"`
	Envelopes string `json:"envelopes" example:"https://example.com/api/v1/envelopes?category=3b1ea324-d438-4419-882a-2fc91d71772f"`
}

type CategoryQueryFilter struct {
	Name     string `form:"name"`
	BudgetID string `form:"budget"`
	Note     string `form:"note"`
}

func (f CategoryQueryFilter) ToCreate(c *gin.Context) (models.CategoryCreate, error) {
	budgetID, err := httputil.UUIDFromString(c, f.BudgetID)
	if err != nil {
		return models.CategoryCreate{}, err
	}

	return models.CategoryCreate{
		Name:     f.Name,
		BudgetID: budgetID,
		Note:     f.Note,
	}, nil
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

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Categories
// @Success     204
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Router      /v1/categories [options]
func (co Controller) OptionsCategoryList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Categories
// @Success     204
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Param       categoryId path string true "ID formatted as string"
// @Router      /v1/categories/{categoryId} [options]
func (co Controller) OptionsCategoryDetail(c *gin.Context) {
	p, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := co.getCategoryObject(c, p)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// @Summary     Create category
// @Description Creates a new category
// @Tags        Categories
// @Produce     json
// @Success     201 {object} CategoryResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Param       category body     models.CategoryCreate true "Category"
// @Router      /v1/categories [post]
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

// @Summary     Get categories
// @Description Returns a list of categories
// @Tags        Categories
// @Produce     json
// @Success     200 {object} CategoryListResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500 {object} httperrors.HTTPError
// @Router      /v1/categories [get]
// @Param       name   query string false "Filter by name"
// @Param       note   query string false "Filter by note"
// @Param       budget query string false "Filter by budget ID"
func (co Controller) GetCategories(c *gin.Context) {
	var filter CategoryQueryFilter

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we are filtering for
	queryFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate(c)
	if err != nil {
		return
	}

	var categories []models.Category
	if !queryWithRetry(c, co.DB.Where(&models.Category{
		CategoryCreate: create,
	}, queryFields...).Find(&categories)) {
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

// @Summary     Get category
// @Description Returns a specific category
// @Tags        Categories
// @Produce     json
// @Success     200 {object} CategoryResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500        {object} httperrors.HTTPError
// @Param       categoryId path     string true "ID formatted as string"
// @Router      /v1/categories/{categoryId} [get]
func (co Controller) GetCategory(c *gin.Context) {
	p, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	categoryObject, ok := co.getCategoryObject(c, p)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, CategoryResponse{Data: categoryObject})
}

// @Summary     Update category
// @Description Update an existing category. Only values to be updated need to be specified.
// @Tags        Categories
// @Accept      json
// @Produce     json
// @Success     200 {object} CategoryResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500        {object} httperrors.HTTPError
// @Param       categoryId path     string                true "ID formatted as string"
// @Param       category   body     models.CategoryCreate true "Category"
// @Router      /v1/categories/{categoryId} [patch]
func (co Controller) UpdateCategory(c *gin.Context) {
	p, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	category, ok := co.getCategoryResource(c, p)
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

// @Summary     Delete category
// @Description Deletes a category
// @Tags        Categories
// @Success     204
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500        {object} httperrors.HTTPError
// @Param       categoryId path     string true "ID formatted as string"
// @Router      /v1/categories/{categoryId} [delete]
func (co Controller) DeleteCategory(c *gin.Context) {
	p, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	category, ok := co.getCategoryResource(c, p)
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
		Model: models.Model{
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

	envelopes, ok := co.getEnvelopeObjects(c, id)
	if !ok {
		return Category{}, false
	}

	return Category{
		resource,
		CategoryLinks{
			Self:      fmt.Sprintf("%s/v1/categories/%s", c.GetString("baseURL"), id),
			Envelopes: fmt.Sprintf("%s/v1/envelopes?category=%s", c.GetString("baseURL"), id),
		},
		envelopes,
	}, true
}
