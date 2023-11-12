package controllers

import (
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// queryAndHandleErrors tries to execute a query. If it fails, it
// tries to reconnect the database and retries the query once.
//
// This function is deprecated. Use `query(tx *gorm.DB) httperrors.Error` and
// perform HTTP responses in the calling method.
func queryAndHandleErrors(c *gin.Context, tx *gorm.DB) bool {
	err := tx.Error
	if err != nil {
		httperrors.Handler(c, err)
		return false
	}

	return true
}

// query executes a query. If an error ocurrs, an appropriate user facing
// error message and status code is returned in an httperrors.Error struct.
func query(c *gin.Context, tx *gorm.DB) httperrors.Error {
	err := tx.Error
	if err != nil {
		return httperrors.Parse(c, err)
	}

	return httperrors.Error{}
}
