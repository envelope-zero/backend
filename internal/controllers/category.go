package controllers

import (
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

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
// @Param        budgetId  path  uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId}/categories [options]
func OptionsCategoryList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Categories
// @Success      204
// @Param        budgetId    path  uint64  true  "ID of the budget"
// @Param        categoryId  path  uint64  true  "ID of the category"
// @Router       /v1/budgets/{budgetId}/categories/{categoryId} [options]
func OptionsCategoryDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// CreateCategory creates a new category.
func CreateCategory(c *gin.Context) {
	var data models.Category

	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	data.BudgetID, _ = strconv.ParseUint(c.Param("budgetId"), 10, 0)
	models.DB.Create(&data)

	c.JSON(http.StatusCreated, gin.H{"data": &data})
}

// GetCategories retrieves all categories.
func GetCategories(c *gin.Context) {
	var categories []models.Category

	// Check if the budget exists at all
	budget, err := getBudget(c)
	if err != nil {
		return
	}

	models.DB.Where(&models.Category{
		BudgetID: budget.ID,
	}).Find(&categories)

	c.JSON(http.StatusOK, gin.H{"data": categories})
}

// GetCategory retrieves a category by its ID.
func GetCategory(c *gin.Context) {
	category, err := getCategory(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": category, "links": map[string]string{
		"envelopes": requestURL(c) + "/envelopes",
	}})
}

// UpdateCategory updates a category, selected by the ID parameter.
func UpdateCategory(c *gin.Context) {
	var category models.Category

	err := models.DB.First(&category, c.Param("categoryId")).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return
	}

	var data models.Category
	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&category).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": category})
}

// DeleteCategory removes a category, identified by its ID.
func DeleteCategory(c *gin.Context) {
	var category models.Category
	err := models.DB.First(&category, c.Param("categoryId")).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&category)

	c.JSON(http.StatusNoContent, gin.H{})
}
