package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/requestid"
	"github.com/rs/zerolog/log"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// RegisterAllocationRoutes registers the routes for allocations with
// the RouterGroup that is passed.
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

// CreateAllocation creates a new allocation.
func CreateAllocation(c *gin.Context) {
	var data models.Allocation

	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	data.EnvelopeID, _ = strconv.ParseUint(c.Param("envelopeId"), 10, 0)
	result := models.DB.Create(&data)

	if result.Error != nil {
		// By default, we assume a server error
		errMessage := "There was an error processing your request, please contact your server administrator"
		status := http.StatusInternalServerError

		// Set helpful error messages for known errors
		if strings.Contains(result.Error.Error(), "UNIQUE constraint failed: allocations.month, allocations.year") {
			errMessage = "You can not create multiple allocations for the same month"
			status = http.StatusBadRequest
		} else if strings.Contains(result.Error.Error(), "CHECK constraint failed: month_valid") {
			errMessage = "The month must be between 1 and 12"
			status = http.StatusBadRequest
		}

		// Print the error to the server log if it’s a server error
		if status == http.StatusInternalServerError {
			log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", result.Error, result.Error.Error())
		}

		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// GetAllocations retrieves all allocations.
func GetAllocations(c *gin.Context) {
	var allocations []models.Allocation

	// Check if the envelope exists
	envelope, err := getEnvelope(c)
	if err != nil {
		return
	}

	models.DB.Where(&models.Allocation{
		EnvelopeID: envelope.ID,
	}).Find(&allocations)

	c.JSON(http.StatusOK, gin.H{"data": allocations})
}

// GetAllocation retrieves a allocation by its ID.
func GetAllocation(c *gin.Context) {
	allocation, err := getAllocation(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": allocation})
}

// UpdateAllocation updates a allocation, selected by the ID parameter.
func UpdateAllocation(c *gin.Context) {
	var allocation models.Allocation

	err := models.DB.First(&allocation, c.Param("allocationId")).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return
	}

	var data models.Allocation
	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&allocation).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": allocation})
}

// DeleteAllocation removes a allocation, identified by its ID.
func DeleteAllocation(c *gin.Context) {
	var allocation models.Allocation
	err := models.DB.First(&allocation, c.Param("allocationId")).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&allocation)

	c.JSON(http.StatusNoContent, gin.H{})
}
