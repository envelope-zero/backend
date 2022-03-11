package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterEnvelopeRoutes registers the routes for envelopes with
// the RouterGroup that is passed
func RegisterEnvelopeRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetEnvelopes)
		r.POST("", CreateEnvelope)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:envelopeId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:envelopeId", GetEnvelope)
		r.PATCH("/:envelopeId", UpdateEnvelope)
		r.DELETE("/:envelopeId", DeleteEnvelope)
	}
}

// CreateEnvelope creates a new envelope
func CreateEnvelope(c *gin.Context) {
	var data models.CreateEnvelope

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	categoryID, _ := strconv.Atoi(c.Param("categoryId"))
	envelope := models.Envelope{Name: data.Name, CategoryID: categoryID}
	models.DB.Create(&envelope)

	c.JSON(http.StatusOK, gin.H{"data": envelope})
}

// GetEnvelopes retrieves all envelopes
func GetEnvelopes(c *gin.Context) {
	var envelopes []models.Envelope
	models.DB.Where("category_id = ?", c.Param("categoryId")).Find(&envelopes)

	c.JSON(http.StatusOK, gin.H{"data": envelopes})
}

// GetEnvelope retrieves a envelope by its ID
func GetEnvelope(c *gin.Context) {
	var envelope models.Envelope
	err := models.DB.First(&envelope, c.Param("envelopeId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": envelope})
}

// UpdateEnvelope updates a envelope, selected by the ID parameter
func UpdateEnvelope(c *gin.Context) {
	var envelope models.Envelope

	err := models.DB.First(&envelope, c.Param("envelopeId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.Envelope
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&envelope).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": envelope})
}

// DeleteEnvelope removes a envelope, identified by its ID
func DeleteEnvelope(c *gin.Context) {
	var envelope models.Envelope
	err := models.DB.First(&envelope, c.Param("envelopeId")).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	models.DB.Delete(&envelope)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
