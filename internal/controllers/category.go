package controllers

import (
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// RegisterCategoryRoutes registers the routes for categories with
// the RouterGroup that is passed.
func RegisterCategoryRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetCategories)
		r.POST("", CreateCategory)
	}

	// Category with ID
	{
		r.OPTIONS("/:categoryId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:categoryId", GetCategory)
		r.PATCH("/:categoryId", UpdateCategory)
		r.DELETE("/:categoryId", DeleteCategory)
	}

	RegisterEnvelopeRoutes(r.Group("/:categoryId/envelopes"))
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
	budgetID, err := checkBudgetExists(c, c.Param("budgetId"))
	if err != nil {
		return
	}

	models.DB.Where(&models.Category{
		BudgetID: budgetID,
	}).Find(&categories)

	c.JSON(http.StatusOK, gin.H{"data": categories})
}

// GetCategory retrieves a category by its ID.
func GetCategory(c *gin.Context) {
	var category models.Category
	err := models.DB.First(&category, c.Param("categoryId")).Error
	if err != nil {
		FetchErrorHandler(c, err)
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
		FetchErrorHandler(c, err)
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
		FetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&category)

	c.JSON(http.StatusNoContent, gin.H{})
}
