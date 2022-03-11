package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterCategoryRoutes registers the routes for categories with
// the RouterGroup that is passed
func RegisterCategoryRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetCategories)
		r.POST("", CreateCategory)
	}

	// Transaction with ID
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

// CreateCategory creates a new category
func CreateCategory(c *gin.Context) {
	var data models.CreateCategory

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budgetID, _ := strconv.Atoi(c.Param("budgetId"))
	category := models.Category{Name: data.Name, BudgetID: budgetID, Note: data.Note}
	models.DB.Create(&category)

	c.JSON(http.StatusOK, gin.H{"data": category})
}

// GetCategories retrieves all categories
func GetCategories(c *gin.Context) {
	var categories []models.Category
	models.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&categories)

	c.JSON(http.StatusOK, gin.H{"data": categories})
}

// GetCategory retrieves a category by its ID
func GetCategory(c *gin.Context) {
	var category models.Category
	err := models.DB.First(&category, c.Param("categoryId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": category, "links": map[string]string{
		"envelopes": "/envelopes",
	}})
}

// UpdateCategory updates a category, selected by the ID parameter
func UpdateCategory(c *gin.Context) {
	var category models.Category

	err := models.DB.First(&category, c.Param("categoryId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.Category
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&category).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": category})
}

// DeleteCategory removes a category, identified by its ID
func DeleteCategory(c *gin.Context) {
	var category models.Category
	err := models.DB.First(&category, c.Param("categoryId")).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	models.DB.Delete(&category)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
