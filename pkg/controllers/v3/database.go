package v3

import (
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// query executes a query. If an error ocurrs, an appropriate user facing
// error message and status code is returned in an httperrors.Error struct.
func query(c *gin.Context, tx *gorm.DB) httperrors.Error {
	err := tx.Error
	if err != nil {
		return httperrors.Parse(c, err)
	}

	return httperrors.Error{}
}
