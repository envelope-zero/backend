package v3

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getModelByID gets a resources of a specified type by its ID.
//
// If the resources does not exist or the ID is the zero UUID, an appropriate error is returned.
func getModelByID[T models.Model](c *gin.Context, id uuid.UUID) (resource T, err httperrors.Error) {
	if id == uuid.Nil {
		return resource, httperrors.Error{Err: fmt.Errorf("no %s ID specified", resource.Self()), Status: http.StatusBadRequest}
	}

	dbErr := models.DB.First(&resource, "id = ?", id).Error
	if dbErr != nil {
		return resource, httperrors.GenericDBError(resource, c, dbErr)
	}

	return resource, httperrors.Error{}
}
