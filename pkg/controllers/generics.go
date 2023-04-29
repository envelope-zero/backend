package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getResourceByID gets a resources of a specified type by its ID.
//
// When the ID is not specified (which is equal to an all-zeroes UUID), it returns an HTTP 400.
// When no resource exists for the specified ID, an HTTP 404 is returned with an appropriate message.
func getResourceByID[T models.Model](c *gin.Context, co Controller, id uuid.UUID) (resource T, success bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "No %s ID specified", resource.Self())
		return
	}

	if !queryWithRetry(c, co.DB.Where(
		map[string]interface{}{"ID": id},
	).First(&resource), fmt.Sprintf("No %s found for the specified ID", resource.Self())) {
		return
	}

	return resource, true
}
