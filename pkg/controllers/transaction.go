package controllers

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// createTransaction creates a single transaction after verifying it is a valid transaction.
func (co Controller) createTransaction(c *gin.Context, t models.Transaction) (models.Transaction, httperrors.ErrorStatus) {
	_, err := getResourceByID[models.Budget](c, co, t.BudgetID)
	if !err.Nil() {
		return t, err
	}

	// Check the source account
	sourceAccount, err := getResourceByID[models.Account](c, co, t.SourceAccountID)
	if !err.Nil() {
		return t, err
	}

	// Check the destination account
	destinationAccount, err := getResourceByID[models.Account](c, co, t.DestinationAccountID)
	if !err.Nil() {
		return t, err
	}

	// Check the transaction
	err = co.checkTransaction(c, t, sourceAccount, destinationAccount)
	if !err.Nil() {
		return t, err
	}

	dbErr := co.DB.Create(&t).Error
	if dbErr != nil {
		return models.Transaction{}, httperrors.GenericDBError[models.Transaction](t, c, dbErr)
	}

	return t, httperrors.ErrorStatus{}
}

// checkTransactionAndHandleErrors verifies that the transaction is correct
//
// It checks that
//   - the transaction is not between two external accounts
//   - if an envelope is set: the transaction is not between two on-budget accounts
//   - if an envelope is set: the envelope exists
//
// It returns true if the transaction is valid, false in all
// other cases.
//
// Deprecated.
func (co Controller) checkTransactionAndHandleErrors(c *gin.Context, transaction models.Transaction, source, destination models.Account) (ok bool) {
	ok = true

	if !decimal.Decimal.IsPositive(transaction.Amount) {
		httperrors.New(c, http.StatusBadRequest, "The transaction amount must be positive")
		return false
	}

	if source.External && destination.External {
		httperrors.New(c, http.StatusBadRequest, "A transaction between two external accounts is not possible.")
		return false
	}

	// Check envelope being set for transfer between on-budget accounts
	if transaction.EnvelopeID != nil {
		if source.OnBudget && destination.OnBudget {
			httperrors.New(c, http.StatusBadRequest, "Transfers between two on-budget accounts must not have an envelope set. Such a transaction would be incoming and outgoing for this envelope at the same time, which is not possible")
			return false
		}
		_, ok = getResourceByIDAndHandleErrors[models.Envelope](c, co, *transaction.EnvelopeID)
	}

	return
}

// checkTransaction verifies that the transaction is correct
//
// It checks that
//   - the transaction is not between two external accounts
//   - if an envelope is set: the transaction is not between two on-budget accounts
//   - if an envelope is set: the envelope exists
func (co Controller) checkTransaction(c *gin.Context, transaction models.Transaction, source, destination models.Account) httperrors.ErrorStatus {
	if !decimal.Decimal.IsPositive(transaction.Amount) {
		return httperrors.ErrorStatus{Err: errors.New("the transaction amount must be positive"), Status: http.StatusBadRequest}
	}

	if source.External && destination.External {
		return httperrors.ErrorStatus{Err: errors.New("a transaction between two external accounts is not possible"), Status: http.StatusBadRequest}
	}

	// Check envelope being set for transfer between on-budget accounts
	if transaction.EnvelopeID != nil {
		if source.OnBudget && destination.OnBudget {
			return httperrors.ErrorStatus{Err: errors.New("transfers between two on-budget accounts must not have an envelope set. Such a transaction would be incoming and outgoing for this envelope at the same time, which is not possible"), Status: http.StatusBadRequest}
		}
		_, err := getResourceByID[models.Envelope](c, co, *transaction.EnvelopeID)
		return err
	}

	return httperrors.ErrorStatus{}
}
