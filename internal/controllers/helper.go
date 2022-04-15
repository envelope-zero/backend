package controllers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
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

func parseID(c *gin.Context, param string) (uint64, error) {
	var parsed uint64

	parsed, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		FetchErrorHandler(c, err)
		return 0, err
	}

	return parsed, nil
}

// getBudget verifies that the budget from the URL parameters exists and returns it.
func getBudget(c *gin.Context) (models.Budget, error) {
	var budget models.Budget

	budgetID, err := parseID(c, "budgetId")
	if err != nil {
		return models.Budget{}, err
	}

	// Check that the budget exists. If not, return a 404
	err = models.DB.Where(&models.Budget{
		Model: models.Model{
			ID: budgetID,
		},
	}).First(&budget).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return models.Budget{}, err
	}

	return budget, nil
}

// getAccount verifies that the request URI is valid for the account and returns it.
func getAccount(c *gin.Context) (models.Account, error) {
	var account models.Account

	budget, err := getBudget(c)
	if err != nil {
		return models.Account{}, err
	}

	accountID, err := parseID(c, "accountId")
	if err != nil {
		return models.Account{}, err
	}

	err = models.DB.First(&account, &models.Account{
		BudgetID: budget.ID,
		Model: models.Model{
			ID: accountID,
		},
	}).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return models.Account{}, err
	}

	return account, nil
}

// getCategory verifies that the category from the URL parameters exists and returns it
//
// It also verifies that the budget that is referred to exists.
func getCategory(c *gin.Context) (models.Category, error) {
	var category models.Category

	categoryID, err := parseID(c, "categoryId")
	if err != nil {
		return models.Category{}, err
	}

	budget, err := getBudget(c)
	if err != nil {
		return models.Category{}, err
	}

	err = models.DB.Where(&models.Category{
		BudgetID: budget.ID,
		Model: models.Model{
			ID: categoryID,
		},
	}).First(&category).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return models.Category{}, err
	}

	return category, nil
}

// getEnvelope verifies that the envelope from the URL parameters exists and returns it
//
// It also verifies that the budget and the category that are referred to exist.
func getEnvelope(c *gin.Context) (models.Envelope, error) {
	var envelope models.Envelope

	envelopeID, err := parseID(c, "envelopeId")
	if err != nil {
		return models.Envelope{}, err
	}

	_, err = getCategory(c)
	if err != nil {
		return models.Envelope{}, err
	}

	err = models.DB.Where(&models.Envelope{
		Model: models.Model{
			ID: envelopeID,
		},
	}).First(&envelope).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return models.Envelope{}, err
	}

	return envelope, nil
}

// getTransaction verifies that the request URI is valid for the transaction and returns it.
func getTransaction(c *gin.Context) (models.Transaction, error) {
	var transaction models.Transaction

	budget, err := getBudget(c)
	if err != nil {
		return models.Transaction{}, err
	}

	accountID, err := parseID(c, "transactionId")
	if err != nil {
		return models.Transaction{}, err
	}

	err = models.DB.First(&transaction, &models.Transaction{
		BudgetID: budget.ID,
		Model: models.Model{
			ID: accountID,
		},
	}).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return models.Transaction{}, err
	}

	return transaction, nil
}

// FetchErrorHandler handles errors for fetching data from the database.
func FetchErrorHandler(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatus(http.StatusNotFound)
	} else if reflect.TypeOf(err) == reflect.TypeOf(&strconv.NumError{}) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "An ID specified in the query string was not a valid uint64",
		})
	} else {
		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf(
				"An error occured on the server during your request, please contact your server administrator. The request id is '%v', send this to your server administrator to help them finding the problem.", requestid.Get(c),
			),
		})
	}
}
