package controllers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterAllocationRoutes registers the routes for allocations with
// the RouterGroup that is passed
func RegisterAllocationRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetAllocations)
		r.POST("", CreateAllocation)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:allocationId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:allocationId", GetAllocation)
		r.PATCH("/:allocationId", UpdateAllocation)
		r.DELETE("/:allocationId", DeleteAllocation)
	}
}

// CreateAllocation creates a new allocation
func CreateAllocation(c *gin.Context) {
	var data models.Allocation

	if status, err := bindData(c, data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	data.EnvelopeID, _ = strconv.Atoi(c.Param("envelopeId"))
	result := models.DB.Create(&data)

	if result.Error != nil {
		log.Println(result.Error)

		errMessage := "There was an error processing your request, please contact your server administrator"
		if result.Error.Error() == "UNIQUE constraint failed: allocations.month, allocations.year" {
			errMessage = "You can not create multiple allocations for the same month"
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": errMessage})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GetAllocations retrieves all allocations
func GetAllocations(c *gin.Context) {
	var allocations []models.Allocation
	models.DB.Where("envelope_id = ?", c.Param("envelopeId")).Find(&allocations)

	c.JSON(http.StatusOK, gin.H{"data": allocations})
}

// GetAllocation retrieves a allocation by its ID
func GetAllocation(c *gin.Context) {
	var allocation models.Allocation
	err := models.DB.First(&allocation, c.Param("allocationId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": allocation})
}

// UpdateAllocation updates a allocation, selected by the ID parameter
func UpdateAllocation(c *gin.Context) {
	var allocation models.Allocation

	err := models.DB.First(&allocation, c.Param("allocationId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.Allocation
	if status, err := bindData(c, data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&allocation).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": allocation})
}

// DeleteAllocation removes a allocation, identified by its ID
func DeleteAllocation(c *gin.Context) {
	var allocation models.Allocation
	err := models.DB.First(&allocation, c.Param("allocationId")).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	models.DB.Delete(&allocation)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
