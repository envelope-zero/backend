package controllers

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// createTransaction creates a single transaction after verifying it is a valid transaction.
func (co Controller) createTransaction(c *gin.Context, t models.Transaction) (models.Transaction, httperrors.Error) {
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

	return t, httperrors.Error{}
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

// ToCreateHandleErrors parses the query string and returns a TransactionCreate struct for
// the database request.
//
// This method is deprecated, use ToCreate() and handle errors in the calling method.
func (f TransactionQueryFilterV1) ToCreateHandleErrors(c *gin.Context) (models.TransactionCreate, bool) {
	budgetID, ok := httputil.UUIDFromStringHandleErrors(c, f.BudgetID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	sourceAccountID, ok := httputil.UUIDFromStringHandleErrors(c, f.SourceAccountID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	destinationAccountID, ok := httputil.UUIDFromStringHandleErrors(c, f.DestinationAccountID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	envelopeID, ok := httputil.UUIDFromStringHandleErrors(c, f.EnvelopeID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	// If the envelopeID is nil, use an actual nil, not uuid.Nil
	var eID *uuid.UUID
	if envelopeID != uuid.Nil {
		eID = &envelopeID
	}

	return models.TransactionCreate{
		Amount:                f.Amount,
		BudgetID:              budgetID,
		SourceAccountID:       sourceAccountID,
		DestinationAccountID:  destinationAccountID,
		EnvelopeID:            eID,
		Reconciled:            f.Reconciled,
		ReconciledSource:      f.ReconciledSource,
		ReconciledDestination: f.ReconciledDestination,
	}, true
}

// ToCreate parses the query string and returns a TransactionCreate struct for
// the database request. On error, it returns httperrors.ErrorStatus struct with.
func (f TransactionQueryFilterV3) ToCreate(c *gin.Context) (models.TransactionCreate, httperrors.Error) {
	budgetID, err := httputil.UUIDFromString(c, f.BudgetID)
	if !err.Nil() {
		return models.TransactionCreate{}, err
	}

	sourceAccountID, err := httputil.UUIDFromString(c, f.SourceAccountID)
	if !err.Nil() {
		return models.TransactionCreate{}, err
	}

	destinationAccountID, err := httputil.UUIDFromString(c, f.DestinationAccountID)
	if !err.Nil() {
		return models.TransactionCreate{}, err
	}

	envelopeID, err := httputil.UUIDFromString(c, f.EnvelopeID)
	if !err.Nil() {
		return models.TransactionCreate{}, err
	}

	// If the envelopeID is nil, use an actual nil, not uuid.Nil
	var eID *uuid.UUID
	if envelopeID != uuid.Nil {
		eID = &envelopeID
	}

	return models.TransactionCreate{
		Amount:                f.Amount,
		BudgetID:              budgetID,
		SourceAccountID:       sourceAccountID,
		DestinationAccountID:  destinationAccountID,
		EnvelopeID:            eID,
		ReconciledSource:      f.ReconciledSource,
		ReconciledDestination: f.ReconciledDestination,
	}, httperrors.Error{}
}
