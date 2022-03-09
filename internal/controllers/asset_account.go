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

// RegisterAssetAccountRoutes registers the routes for accounts with
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
		r.OPTIONS("/:accountId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:accountId", GetAssetAccount)
		r.PATCH("/:accountId", UpdateAssetAccount)
		r.DELETE("/:accountId", DeleteAssetAccount)
	}
}

// CreateAssetAccount creates a new account
func CreateAssetAccount(c *gin.Context) {
	var data models.CreateAssetAccount

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budgetID, _ := strconv.Atoi(c.Param("budgetId"))
	account := models.AssetAccount{Name: data.Name, BudgetID: budgetID}
	database.DB.Create(&account)

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// GetAssetAccounts retrieves all accounts
func GetAssetAccounts(c *gin.Context) {
	var accounts []models.AssetAccount
	database.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&accounts)

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

// GetAssetAccount retrieves a account by its ID
func GetAssetAccount(c *gin.Context) {
	var account models.AssetAccount
	err := database.DB.First(&account, c.Param("accountId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// UpdateAssetAccount updates a account, selected by the ID parameter
func UpdateAssetAccount(c *gin.Context) {
	var account models.AssetAccount

	err := database.DB.First(&account, c.Param("accountId")).Error

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

	database.DB.Model(&account).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": account})
}

// DeleteAssetAccount removes a account, identified by its ID
func DeleteAssetAccount(c *gin.Context) {
	var account models.AssetAccount
	err := database.DB.First(&account, c.Param("accountId")).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	database.DB.Delete(&account)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
