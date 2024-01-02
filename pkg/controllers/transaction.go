package controllers

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// createTransaction creates a single transaction after verifying it is a valid transaction.
func (co Controller) createTransaction(c *gin.Context, model *models.Transaction) httperrors.Error {
	_, err := getResourceByID[models.Budget](c, co, model.BudgetID)
	if !err.Nil() {
		return err
	}

	// Check the source account
	sourceAccount, err := getResourceByID[models.Account](c, co, model.SourceAccountID)
	if !err.Nil() {
		return err
	}

	// Check the destination account
	destinationAccount, err := getResourceByID[models.Account](c, co, model.DestinationAccountID)
	if !err.Nil() {
		return err
	}

	// Check the transaction
	err = co.checkTransaction(c, *model, sourceAccount, destinationAccount)
	if !err.Nil() {
		return err
	}

	dbErr := co.DB.Create(&model).Error
	if dbErr != nil {
		return httperrors.GenericDBError[models.Transaction](models.Transaction{}, c, dbErr)
	}

	return httperrors.Error{}
}

// checkTransaction verifies that the transaction is correct
//
// It checks that
//   - the transaction is not between two external accounts
//   - if an envelope is set: the transaction is not between two on-budget accounts
//   - if an envelope is set: the envelope exists
func (co Controller) checkTransaction(c *gin.Context, transaction models.Transaction, source, destination models.Account) httperrors.Error {
	if !decimal.Decimal.IsPositive(transaction.Amount) {
		return httperrors.Error{Err: errors.New("the transaction amount must be positive"), Status: http.StatusBadRequest}
	}

	if source.External && destination.External {
		return httperrors.Error{Err: errors.New("a transaction between two external accounts is not possible"), Status: http.StatusBadRequest}
	}

	// Check envelope being set for transfer between on-budget accounts
	if transaction.EnvelopeID != nil && *transaction.EnvelopeID != uuid.Nil {
		if source.OnBudget && destination.OnBudget {
			// TODO: Verify this state in the model hooks
			return httperrors.Error{Err: errors.New("transfers between two on-budget accounts must not have an envelope set. Such a transaction would be incoming and outgoing for this envelope at the same time, which is not possible"), Status: http.StatusBadRequest}
		}
		_, err := getResourceByID[models.Envelope](c, co, *transaction.EnvelopeID)
		return err
	}

	return httperrors.Error{}
}
