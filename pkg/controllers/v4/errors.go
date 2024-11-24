package v4

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v5/pkg/models"
)

type httpError struct {
	Error string `json:"error" example:"An ID specified in the query string was not a valid UUID"`
}

// status returns the appropriate status for a database error
func status(err error) int {
	if errors.Is(err, models.ErrGeneral) {
		return http.StatusInternalServerError
	}

	if errors.Is(err, models.ErrResourceNotFound) {
		return http.StatusNotFound
	}

	return http.StatusBadRequest
}

var (
	errAccountIDParameter = errors.New("the accountId parameter must be set")
	errMonthNotSetInQuery = errors.New("the month query parameter must be set")
)

// Cleanup errors
var (
	errCleanupConfirmation = errors.New("the confirmation for the cleanup API call was incorrect")
)

// Import errors
var (
	errNoFilePost       = errors.New("you must send a file to this endpoint")
	errWrongFileSuffix  = errors.New("this endpoint only supports files of the following types")
	errBudgetNameInUse  = errors.New("this budget name is already in use. Imports from YNAB 4 create a new budget, therefore the name needs to be unique")
	errBudgetNameNotSet = errors.New("the budgetName parameter must be set")
)

// Transaction errors
var (
	errTransactionDirectionInvalid = errors.New("the specified transaction direction is invalid")
	errTransactionTypeInvalid      = errors.New("the specified transaction type is invalid")
)
