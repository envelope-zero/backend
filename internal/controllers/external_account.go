package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterExternalAccountRoutes registers the routes for externalAccounts with
// the RouterGroup that is passed
func RegisterExternalAccountRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetExternalAccounts)
		r.POST("", CreateExternalAccount)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:externalAccountId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:externalAccountId", GetExternalAccount)
		r.PATCH("/:externalAccountId", UpdateExternalAccount)
		r.DELETE("/:externalAccountId", DeleteExternalAccount)
	}
}

// CreateExternalAccount creates a new externalAccount
func CreateExternalAccount(c *gin.Context) {
	var data models.CreateExternalAccount

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budgetID, _ := strconv.Atoi(c.Param("budgetId"))
	externalAccount := models.ExternalAccount{Name: data.Name, BudgetID: budgetID}
	database.DB.Create(&externalAccount)

	c.JSON(http.StatusOK, gin.H{"data": externalAccount})
}

// GetExternalAccounts retrieves all externalAccounts
func GetExternalAccounts(c *gin.Context) {
	var externalAccounts []models.ExternalAccount
	database.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&externalAccounts)

	c.JSON(http.StatusOK, gin.H{"data": externalAccounts})
}

// GetExternalAccount retrieves an externalAccount by its ID
func GetExternalAccount(c *gin.Context) {
	var externalAccount models.ExternalAccount
	err := database.DB.First(&externalAccount, c.Param("externalAccountId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": externalAccount})
}

// UpdateExternalAccount updates an externalAccount, selected by the ID parameter
func UpdateExternalAccount(c *gin.Context) {
	var externalAccount models.ExternalAccount

	err := database.DB.First(&externalAccount, c.Param("externalAccountId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.ExternalAccount
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&externalAccount).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": externalAccount})
}

// DeleteExternalAccount removes an externalAccount, identified by its ID
func DeleteExternalAccount(c *gin.Context) {
	var externalAccount models.ExternalAccount
	err := database.DB.First(&externalAccount, c.Param("externalAccountId")).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	database.DB.Delete(&externalAccount)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
