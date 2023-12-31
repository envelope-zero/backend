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
func (co Controller) createTransaction(c *gin.Context, create models.TransactionCreate) (models.Transaction, httperrors.Error) {
	t := models.Transaction{
		TransactionCreate: create,
	}

	_, err := getResourceByID[models.Budget](c, co, t.BudgetID)
	if !err.Nil() {
		return models.Transaction{}, err
	}

	// Check the source account
	sourceAccount, err := getResourceByID[models.Account](c, co, t.SourceAccountID)
	if !err.Nil() {
		return models.Transaction{}, err
	}

	// Check the destination account
	destinationAccount, err := getResourceByID[models.Account](c, co, t.DestinationAccountID)
	if !err.Nil() {
		return models.Transaction{}, err
	}

	// Check the transaction
	err = co.checkTransaction(c, t, sourceAccount, destinationAccount)
	if !err.Nil() {
		return models.Transaction{}, err
	}

	dbErr := co.DB.Create(&t).Error
	if dbErr != nil {
		return models.Transaction{}, httperrors.GenericDBError[models.Transaction](t, c, dbErr)
	}

	return t, httperrors.Error{}
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

// ToCreate parses the query string and returns a TransactionCreate struct for
// the database request. On error, it returns httperrors.ErrorStatus struct with.
func (f TransactionQueryFilterV3) ToCreate() (models.TransactionCreate, httperrors.Error) {
	budgetID, err := httputil.UUIDFromString(f.BudgetID)
	if !err.Nil() {
		return models.TransactionCreate{}, err
	}

	sourceAccountID, err := httputil.UUIDFromString(f.SourceAccountID)
	if !err.Nil() {
		return models.TransactionCreate{}, err
	}

	destinationAccountID, err := httputil.UUIDFromString(f.DestinationAccountID)
	if !err.Nil() {
		return models.TransactionCreate{}, err
	}

	envelopeID, err := httputil.UUIDFromString(f.EnvelopeID)
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
