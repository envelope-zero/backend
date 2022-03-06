package controllers

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateBudget creates a new budget
func CreateBudget(c *gin.Context) {
	var data models.CreateBudget

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budget := models.Budget{Name: data.Name}
	database.DB.Create(&budget)

	c.JSON(http.StatusOK, gin.H{"data": budget})
}

// GetBudgets retrieves all budgets
func GetBudgets(c *gin.Context) {
	var budgets []models.Budget
	database.DB.Find(&budgets)

	c.JSON(http.StatusOK, gin.H{"data": budgets})
}

// GetBudget retrieves a budget by its ID
func GetBudget(c *gin.Context) {
	var budget models.Budget
	err := database.DB.First(&budget, c.Param("id")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": budget})
}

// UpdateBudget updates a budget, selected by the ID parameter
func UpdateBudget(c *gin.Context) {
	var budget models.Budget

	err := database.DB.First(&budget, c.Param("id")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.Budget
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&budget).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": budget})
}

// DeleteBudget removes a budget, identified by its ID
func DeleteBudget(c *gin.Context) {
	var budget models.Budget
	err := database.DB.First(&budget, c.Param("id")).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	database.DB.Delete(&budget)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
