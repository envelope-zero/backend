package controllers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-contrib/requestid"
	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// bindData binds the data from the request to the struct passed in the interface.
func bindData(c *gin.Context, data interface{}) (int, error) {
	if err := c.ShouldBindJSON(&data); err != nil {
		if errors.Is(io.EOF, err) {
			return http.StatusBadRequest, errors.New("request body must not be emtpy")
		}

		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		return http.StatusBadRequest, errors.New("the body of your request contains invalid or un-parseable data. Please check and try again")
	}
	return http.StatusOK, nil
}

// requestURL returns the full request URL.
//
// The scheme defaults to https and only falls back to http
// if the x-forwarded-proto header is set to "http".
//
func requestURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.Header.Get("x-forwarded-proto") == "https" {
		scheme = "https"
	}

	forwardedPrefix := c.Request.Header.Get("x-forwarded-prefix")
	return scheme + "://" + c.Request.Host + forwardedPrefix + c.Request.URL.Path
}

func checkBudgetExists(c *gin.Context, budgetIDString string) (uint64, error) {
	var budget models.Budget

	budgetID, _ := strconv.ParseUint(budgetIDString, 10, 64)

	// Check that the budget exists. If not, return a 404
	err := models.DB.Where(&models.Budget{
		Model: models.Model{
			ID: budgetID,
		},
	}).First(&budget).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return 0, err
	}

	return budgetID, nil
}

// FetchErrorHandler handles errors for fetching data from the database.
func FetchErrorHandler(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusNotFound)
	} else {
		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf(
				"An error occured on the server during your request, please contact your server administrator. The request id is '%v', send this to your server administrator to help them finding the problem.", requestid.Get(c),
			),
		})
	}
}
