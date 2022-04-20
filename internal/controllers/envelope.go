package controllers

import (
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// RegisterEnvelopeRoutes registers the routes for envelopes with
// the RouterGroup that is passed.
func RegisterEnvelopeRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetEnvelopes)
		r.POST("", CreateEnvelope)
	}

	// Envelope with ID
	{
		r.OPTIONS("/:envelopeId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:envelopeId", GetEnvelope)
		r.PATCH("/:envelopeId", UpdateEnvelope)
		r.DELETE("/:envelopeId", DeleteEnvelope)
	}

	RegisterAllocationRoutes(r.Group("/:envelopeId/allocations"))
}

// CreateEnvelope creates a new envelope.
func CreateEnvelope(c *gin.Context) {
	var data models.Envelope

	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	data.CategoryID, _ = strconv.ParseUint(c.Param("categoryId"), 10, 0)
	models.DB.Create(&data)

	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// GetEnvelopes retrieves all envelopes.
func GetEnvelopes(c *gin.Context) {
	var envelopes []models.Envelope

	// Check if the category exists at all
	category, err := getCategory(c)
	if err != nil {
		return
	}

	models.DB.Where(&models.Envelope{
		CategoryID: category.ID,
	}).Find(&envelopes)

	c.JSON(http.StatusOK, gin.H{"data": envelopes})
}

// GetEnvelope retrieves a envelope by its ID.
func GetEnvelope(c *gin.Context) {
	envelope, err := getEnvelope(c)
	if err != nil {
		return
	}

	// Parse the month from the request
	var month Month
	if err := c.ShouldBind(&month); err != nil {
		FetchErrorHandler(c, err)
		return
	}

	// If a month is requested, return only month specfic data
	if !month.Month.IsZero() {
		c.JSON(http.StatusOK, gin.H{
			"data": envelope.Month(month.Month),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": envelope,
		"links": map[string]string{
			"allocations": requestURL(c) + "/allocations",
			"month":       requestURL(c) + "?month=YYYY-MM",
		},
	})
}

// UpdateEnvelope updates a envelope, selected by the ID parameter.
func UpdateEnvelope(c *gin.Context) {
	var envelope models.Envelope

	err := models.DB.First(&envelope, c.Param("envelopeId")).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return
	}

	var data models.Envelope
	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&envelope).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": envelope})
}

// DeleteEnvelope removes a envelope, identified by its ID.
func DeleteEnvelope(c *gin.Context) {
	var envelope models.Envelope
	err := models.DB.First(&envelope, c.Param("envelopeId")).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&envelope)

	c.JSON(http.StatusNoContent, gin.H{})
}
