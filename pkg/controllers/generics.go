package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getResourceByIDAndHandleErrors gets a resources of a specified type by its ID.
//
// When the ID is not specified (which is equal to an all-zeroes UUID), it returns an HTTP 400.
// When no resource exists for the specified ID, an HTTP 404 is returned with an appropriate message.
func getResourceByIDAndHandleErrors[T models.Model](c *gin.Context, co Controller, id uuid.UUID) (resource T, success bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "No %s ID specified", resource.Self())
		return
	}

	err := query(c, co.DB.Where(
		map[string]interface{}{"ID": id},
	).First(&resource))
	if !err.Nil() {
		msg := err.Error()
		if err.Status == http.StatusNotFound {
			s := fmt.Sprintf("No %s found for the specified ID", resource.Self())
			msg = s
		}

		c.JSON(err.Status, httperrors.HTTPError{
			Error: msg,
		})
		return resource, false
	}

	return resource, true
}

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
