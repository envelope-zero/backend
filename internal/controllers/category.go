package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type CategoryListResponse struct {
	Data []models.Category `json:"data"`
}

type CategoryResponse struct {
	Data  models.Category `json:"data"`
	Links CategoryLinks   `json:"links"`
}

type CategoryLinks struct {
	Envelopes string `json:"envelopes" example:"https://example.com/api/v1/budgets/5/categories/7/envelopes"`
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

	RegisterEnvelopeRoutes(r.Group("/:categoryId/envelopes"))
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Categories
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Param        budgetId    path      uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId}/categories [options]
func OptionsCategoryList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Categories
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Param        budgetId    path  uint64  true  "ID of the budget"
// @Param        categoryId  path  uint64  true  "ID of the category"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId} [options]
func OptionsCategoryDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// @Summary      Create category
// @Description  Create a new category for a specific budget
// @Tags         Categories
// @Produce      json
// @Success      201  {object}  CategoryResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500       {object}  httputil.HTTPError
// @Param        budgetId  path      uint64                 true  "ID of the budget"
// @Param        category    body      models.CategoryCreate  true  "Category"
// @Router       /v1/budgets/{budgetId}/categories [post]
func CreateCategory(c *gin.Context) {
	var data models.Category

	status, err := httputil.BindData(c, &data)
	if err != nil {
		httputil.NewError(c, status, err)
		return
	}

	data.CategoryCreate.BudgetID, err = httputil.ParseID(c, "budgetId")
	if err != nil {
		return
	}
	models.DB.Create(&data)
	c.JSON(http.StatusCreated, CategoryResponse{Data: data})
}

// @Summary      Get all categories for a budget
// @Description  Returns the full list of all categories for a specific budget
// @Tags         Categories
// @Produce      json
// @Success      200  {object}  CategoryListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500       {object}  httputil.HTTPError
// @Param        budgetId    path      uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId}/categories [get]
func GetCategories(c *gin.Context) {
	categories, err := getCategoryResources(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, CategoryListResponse{Data: categories})
}

// @Summary      Get category
// @Description  Returns a category by its ID
// @Tags         Categories
// @Produce      json
// @Success      200  {object}  CategoryResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId  path      uint64  true  "ID of the budget"
// @Param        categoryId  path      uint64  true  "ID of the category"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId} [get]
func GetCategory(c *gin.Context) {
	_, err := getCategoryResource(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, newCategoryResponse(c))
}

// @Summary      Update a category
// @Description  Update an existing category. Only values to be updated need to be specified.
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Success      200  {object}  CategoryResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId    path      uint64                 true  "ID of the budget"
// @Param        categoryId  path      uint64                 true  "ID of the category"
// @Param        category  body      models.CategoryCreate  true  "Category"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId} [patch]
func UpdateCategory(c *gin.Context) {
	category, err := getCategoryResource(c)
	if err != nil {
		return
	}

	var data models.Category
	if status, err := httputil.BindData(c, &data); err != nil {
		httputil.NewError(c, status, err)
		return
	}

	models.DB.Model(&category).Updates(data)
	c.JSON(http.StatusOK, CategoryResponse{Data: category})
}

// @Summary      Delete a category
// @Description  Deletes an existing category
// @Tags         Categories
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500         {object}  httputil.HTTPError
// @Param        budgetId  path  uint64  true  "ID of the budget"
// @Param        categoryId  path      uint64  true  "ID of the category"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId} [delete]
func DeleteCategory(c *gin.Context) {
	category, err := getCategoryResource(c)
	if err != nil {
		return
	}

	models.DB.Delete(&category)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getCategoryResource verifies that the category from the URL parameters exists and returns it.
func getCategoryResource(c *gin.Context) (models.Category, error) {
	var category models.Category

	categoryID, err := httputil.ParseID(c, "categoryId")
	if err != nil {
		return models.Category{}, err
	}

	budget, err := getBudgetResource(c)
	if err != nil {
		return models.Category{}, err
	}

	err = models.DB.Where(&models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: budget.ID,
		},
		Model: models.Model{
			ID: categoryID,
		},
	}).First(&category).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Category{}, err
	}

	return category, nil
}

// getCategoryResources returns all categories for the requested budget.
func getCategoryResources(c *gin.Context) ([]models.Category, error) {
	var categories []models.Category

	budget, err := getBudgetResource(c)
	if err != nil {
		return []models.Category{}, err
	}

	models.DB.Where(&models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: budget.ID,
		},
	}).Find(&categories)

	return categories, nil
}

func newCategoryResponse(c *gin.Context) CategoryResponse {
	// When this function is called, all parent resources have already been validated
	budget, _ := getBudgetResource(c)
	category, _ := getCategoryResource(c)

	url := httputil.RequestPathV1(c) + fmt.Sprintf("/budgets/%d/categories/%d", budget.ID, category.ID)

	return CategoryResponse{
		Data: category,
		Links: CategoryLinks{
			Envelopes: url + "/envelopes",
		},
	}
}
