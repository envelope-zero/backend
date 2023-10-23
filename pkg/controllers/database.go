package controllers

import (
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// queryAndHandleErrors tries to execute a query. If it fails, it
// tries to reconnect the database and retries the query once.
func queryAndHandleErrors(c *gin.Context, tx *gorm.DB, notFoundMsg ...string) bool {
	err := tx.Error
	if err != nil {
		httperrors.Handler(c, err, notFoundMsg...)
		return false
	}

	return true
}
