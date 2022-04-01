package controllers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// bindData binds the data from the request to the struct passed in the interface.
func bindData(c *gin.Context, data interface{}) (int, error) {
	if err := c.ShouldBindJSON(&data); err != nil {
		if errors.Is(io.EOF, err) {
			return http.StatusBadRequest, errors.New("request body must not be emtpy")
		}

		log.Println(err)
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

	return scheme + "://" + c.Request.Host + c.Request.URL.Path
}

// fetchErrorHandler handles errors for fetching data from the database.
func fetchErrorHandler(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
	} else {
		log.Println(fmt.Sprintf("%T: %v", err, err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occured on the server during your request, please contact your server administrator."})
	}
}
