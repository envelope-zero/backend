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

// RegisterAssetAccountRoutes registers the routes for assetAccounts with
// the RouterGroup that is passed
func RegisterAssetAccountRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetAssetAccounts)
		r.POST("", CreateAssetAccount)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:assetAccountId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:assetAccountId", GetAssetAccount)
		r.PATCH("/:assetAccountId", UpdateAssetAccount)
		r.DELETE("/:assetAccountId", DeleteAssetAccount)
	}
}

// CreateAssetAccount creates a new assetAccount
func CreateAssetAccount(c *gin.Context) {
	var data models.CreateAssetAccount

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budgetID, _ := strconv.Atoi(c.Param("budgetId"))
	assetAccount := models.AssetAccount{Name: data.Name, BudgetID: budgetID}
	database.DB.Create(&assetAccount)

	c.JSON(http.StatusOK, gin.H{"data": assetAccount})
}

// GetAssetAccounts retrieves all assetAccounts
func GetAssetAccounts(c *gin.Context) {
	var assetAccounts []models.AssetAccount
	database.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&assetAccounts)

	c.JSON(http.StatusOK, gin.H{"data": assetAccounts})
}

// GetAssetAccount retrieves an assetAccount by its ID
func GetAssetAccount(c *gin.Context) {
	var assetAccount models.AssetAccount
	err := database.DB.First(&assetAccount, c.Param("assetAccountId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": assetAccount})
}

// UpdateAssetAccount updates an assetAccount, selected by the ID parameter
func UpdateAssetAccount(c *gin.Context) {
	var assetAccount models.AssetAccount

	err := database.DB.First(&assetAccount, c.Param("assetAccountId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.AssetAccount
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&assetAccount).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": assetAccount})
}

// DeleteAssetAccount removes a assetAccount, identified by its ID
func DeleteAssetAccount(c *gin.Context) {
	var assetAccount models.AssetAccount
	err := database.DB.First(&assetAccount, c.Param("assetAccountId")).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	database.DB.Delete(&assetAccount)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
