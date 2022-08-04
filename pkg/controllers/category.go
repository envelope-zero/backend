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

// RegisterCategoryRoutes registers the routes for categories with
// the RouterGroup that is passed.
func RegisterCategoryRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsCategoryList)
		r.GET("", GetCategories)
		r.POST("", CreateCategory)
	}

	// Category with ID
	{
		r.OPTIONS("/:categoryId", OptionsCategoryDetail)
		r.GET("/:categoryId", GetCategory)
		r.PATCH("/:categoryId", UpdateCategory)
		r.DELETE("/:categoryId", DeleteCategory)
	}
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Categories
// @Success     204
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Router      /v1/categories [options]
func OptionsCategoryList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Categories
// @Success     204
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Param       categoryId path string true "ID formatted as string"
// @Router      /v1/categories/{categoryId} [options]
func OptionsCategoryDetail(c *gin.Context) {
	p, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	_, err = getCategoryObject(c, p)
	if err != nil {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// @Summary     Create category
// @Description Creates a new category
// @Tags        Categories
// @Produce     json
// @Success     201 {object} CategoryResponse
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500      {object} httputil.HTTPError
// @Param       category body     models.CategoryCreate true "Category"
// @Router      /v1/categories [post]
func CreateCategory(c *gin.Context) {
	var category models.Category

	err := httputil.BindData(c, &category)
	if err != nil {
		return
	}

	_, err = getBudgetResource(c, category.BudgetID)
	if err != nil {
		return
	}

	database.DB.Create(&category)

	categoryObject, _ := getCategoryObject(c, category.ID)
	c.JSON(http.StatusCreated, CategoryResponse{Data: categoryObject})
}

// @Summary     Get categories
// @Description Returns a list of categories
// @Tags        Categories
// @Produce     json
// @Success     200 {object} CategoryListResponse
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500 {object} httputil.HTTPError
// @Router      /v1/categories [get]
func GetCategories(c *gin.Context) {
	var categories []models.Category

	database.DB.Find(&categories)

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	categoryObjects := make([]Category, 0)

	for _, category := range categories {
		o, _ := getCategoryObject(c, category.ID)
		categoryObjects = append(categoryObjects, o)
	}

	c.JSON(http.StatusOK, CategoryListResponse{Data: categoryObjects})
}

// @Summary     Get category
// @Description Returns a specific category
// @Tags        Categories
// @Produce     json
// @Success     200 {object} CategoryResponse
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500        {object} httputil.HTTPError
// @Param       categoryId path     string true "ID formatted as string"
// @Router      /v1/categories/{categoryId} [get]
func GetCategory(c *gin.Context) {
	p, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	categoryObject, err := getCategoryObject(c, p)
	if err != nil {
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
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500        {object} httputil.HTTPError
// @Param       categoryId path     string                true "ID formatted as string"
// @Param       category   body     models.CategoryCreate true "Category"
// @Router      /v1/categories/{categoryId} [patch]
func UpdateCategory(c *gin.Context) {
	p, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	category, err := getCategoryResource(c, p)
	if err != nil {
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

	err = database.DB.Model(&category).Select("", updateFields...).Updates(data).Error
	if err != nil {
		httputil.ErrorHandler(c, err)
		return
	}

	categoryObject, _ := getCategoryObject(c, category.ID)
	c.JSON(http.StatusOK, CategoryResponse{Data: categoryObject})
}

// @Summary     Delete category
// @Description Deletes a category
// @Tags        Categories
// @Success     204
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500        {object} httputil.HTTPError
// @Param       categoryId path     string true "ID formatted as string"
// @Router      /v1/categories/{categoryId} [delete]
func DeleteCategory(c *gin.Context) {
	p, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	category, err := getCategoryResource(c, p)
	if err != nil {
		return
	}

	database.DB.Delete(&category)

	c.JSON(http.StatusNoContent, gin.H{})
}

func getCategoryResource(c *gin.Context, id uuid.UUID) (models.Category, error) {
	if id == uuid.Nil {
		err := errors.New("No category ID specified")
		httputil.NewError(c, http.StatusBadRequest, err)
		return models.Category{}, err
	}

	var category models.Category

	err := database.DB.Where(&models.Category{
		Model: models.Model{
			ID: id,
		},
	}).First(&category).Error
	if err != nil {
		httputil.ErrorHandler(c, err)
		return models.Category{}, err
	}

	return category, nil
}

// getCategoryResources returns all categories for the requested budget.
func getCategoryResources(c *gin.Context, id uuid.UUID) ([]models.Category, error) {
	var categories []models.Category

	database.DB.Where(&models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: id,
		},
	}).Find(&categories)

	return categories, nil
}

func getCategoryObject(c *gin.Context, id uuid.UUID) (Category, error) {
	resource, err := getCategoryResource(c, id)
	if err != nil {
		return Category{}, err
	}

	envelopes, err := getEnvelopeObjects(c, id)
	if err != nil {
		return Category{}, err
	}

	return Category{
		resource,
		getCategoryLinks(c, id),
		envelopes,
	}, nil
}

// getCategoryLinks returns a BudgetLinks struct.
//
// This function is only needed for getCategoryObject as we cannot create an instance of Category
// with mixed named and unnamed parameters.
func getCategoryLinks(c *gin.Context, id uuid.UUID) CategoryLinks {
	url := fmt.Sprintf("%s/v1/categories/%s", c.GetString("baseURL"), id)

	return CategoryLinks{
		Self:      url,
		Envelopes: fmt.Sprintf("%s/v1/envelopes?category=%s", c.GetString("baseURL"), id),
	}
}
