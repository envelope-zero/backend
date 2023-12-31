package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getResourceByID gets a resources of a specified type by its ID.
//
// If the resources does not exist or the ID is the zero UUID, an appropriate error is returned.
func getResourceByID[T models.Model](c *gin.Context, co Controller, id uuid.UUID) (resource T, err httperrors.Error) {
	if id == uuid.Nil {
		return resource, httperrors.Error{Err: fmt.Errorf("no %s ID specified", resource.Self()), Status: http.StatusBadRequest}
	}

	dbErr := co.DB.First(&resource, "id = ?", id).Error
	if dbErr != nil {
		return resource, httperrors.GenericDBError(resource, c, dbErr)
	}

	return resource, httperrors.Error{}
}
