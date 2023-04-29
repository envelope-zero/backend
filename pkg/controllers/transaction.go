package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/envelope-zero/backend/v2/pkg/httputil"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

type TransactionListResponse struct {
	Data []models.Transaction `json:"data"`
}

type TransactionResponse struct {
	Data models.Transaction `json:"data"`
}

type TransactionQueryFilter struct {
	Date                  time.Time       `form:"date" filterField:"false"`
	FromDate              time.Time       `form:"fromDate" filterField:"false"`
	UntilDate             time.Time       `form:"untilDate" filterField:"false"`
	Amount                decimal.Decimal `form:"amount"`
	AmountLessOrEqual     decimal.Decimal `form:"amountLessOrEqual" filterField:"false"` // Amount less than or equal to this
	AmountMoreOrEqual     decimal.Decimal `form:"amountMoreOrEqual" filterField:"false"` // Amount more than or equal to this
	Note                  string          `form:"note" filterField:"false"`
	BudgetID              string          `form:"budget"`
	SourceAccountID       string          `form:"source"`
	DestinationAccountID  string          `form:"destination"`
	EnvelopeID            string          `form:"envelope"`
	Reconciled            bool            `form:"reconciled"`            // DEPRECATED. Do not use, this field does not work as intended. See https://github.com/envelope-zero/backend/issues/528. Use reconciledSource and reconciledDestination instead.
	ReconciledSource      bool            `form:"reconciledSource"`      // Is the transaction reconciled in the source account?
	ReconciledDestination bool            `form:"reconciledDestination"` // Is the transaction reconciled in the destination account?
	AccountID             string          `form:"account" filterField:"false"`
}

func (f TransactionQueryFilter) ToCreate(c *gin.Context) (models.TransactionCreate, bool) {
	budgetID, ok := httputil.UUIDFromString(c, f.BudgetID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	sourceAccountID, ok := httputil.UUIDFromString(c, f.SourceAccountID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	destinationAccountID, ok := httputil.UUIDFromString(c, f.DestinationAccountID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	envelopeID, ok := httputil.UUIDFromString(c, f.EnvelopeID)
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

// RegisterTransactionRoutes registers the routes for transactions with
// the RouterGroup that is passed.
func (co Controller) RegisterTransactionRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsTransactionList)
		r.GET("", co.GetTransactions)
		r.POST("", co.CreateTransaction)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:transactionId", co.OptionsTransactionDetail)
		r.GET("/:transactionId", co.GetTransaction)
		r.PATCH("/:transactionId", co.UpdateTransaction)
		r.DELETE("/:transactionId", co.DeleteTransaction)
	}
}

// OptionsTransactionList returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Transactions
//	@Success		204
//	@Router			/v1/transactions [options]
func (co Controller) OptionsTransactionList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsTransactionDetail returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Transactions
//	@Success		204
//	@Param			transactionId	path	string	true	"ID formatted as string"
//	@Router			/v1/transactions/{transactionId} [options]
func (co Controller) OptionsTransactionDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var t models.Transaction
	err = co.DB.First(&t, id).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// CreateTransaction creates a new transaction
//
//	@Summary		Create transaction
//	@Description	Creates a new transaction
//	@Tags			Transactions
//	@Produce		json
//	@Success		201	{object}	TransactionResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			transaction	body		models.TransactionCreate	true	"Transaction"
//	@Router			/v1/transactions [post]
func (co Controller) CreateTransaction(c *gin.Context) {
	var transaction models.Transaction

	if err := httputil.BindData(c, &transaction); err != nil {
		return
	}

	// Check if the budget that the transaction shoud belong to exists
	_, ok := getResourceByID[models.Budget](c, co, transaction.BudgetID)
	if !ok {
		return
	}

	// Check the source account
	sourceAccount, ok := getResourceByID[models.Account](c, co, transaction.SourceAccountID)
	if !ok {
		return
	}

	// Check the destination account
	destinationAccount, ok := getResourceByID[models.Account](c, co, transaction.DestinationAccountID)
	if !ok {
		return
	}

	// Check the transaction
	if !co.checkTransaction(c, transaction, sourceAccount, destinationAccount) {
		return
	}

	if !queryWithRetry(c, co.DB.Create(&transaction)) {
		return
	}

	// TODO: Delete
	// err := co.DB.First(&transaction, transaction.ID).Error
	// if err != nil {
	// 	httperrors.Handler(c, err)
	// 	return
	// }
	c.JSON(http.StatusCreated, TransactionResponse{Data: transaction})
}

// GetTransactions returns transactions filtered by the query parameters
//
//	@Summary		Get transactions
//	@Description	Returns a list of transactions
//	@Tags			Transactions
//	@Produce		json
//	@Success		200	{object}	TransactionListResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/transactions [get]
//	@Param			date					query	string	false	"Date of the transaction. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
//	@Param			fromDate				query	string	false	"Transactions at and after this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
//	@Param			untilDate				query	string	false	"Transactions before and at this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
//	@Param			amount					query	string	false	"Filter by amount"
//	@Param			amountLessOrEqual		query	string	false	"Amount less than or equal to this"
//	@Param			amountMoreOrEqual		query	string	false	"Amount more than or equal to this"
//	@Param			note					query	string	false	"Filter by note"
//	@Param			budget					query	string	false	"Filter by budget ID"
//	@Param			account					query	string	false	"Filter by ID of associated account, regardeless of source or destination"
//	@Param			source					query	string	false	"Filter by source account ID"
//	@Param			destination				query	string	false	"Filter by destination account ID"
//	@Param			envelope				query	string	false	"Filter by envelope ID"
//	@Param			reconciled				query	bool	false	"DEPRECATED. Filter by reconcilication state"
//	@Param			reconciledSource		query	bool	false	"Reconcilication state in source account"
//	@Param			reconciledDestination	query	bool	false	"Reconcilication state in destination account"
func (co Controller) GetTransactions(c *gin.Context) {
	var filter TransactionQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the fields set in the filter
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, ok := filter.ToCreate(c)
	if !ok {
		return
	}

	var query *gorm.DB
	query = co.DB.Order("date(date) DESC").Where(&models.Transaction{
		TransactionCreate: create,
	}, queryFields...)

	// Filter for the transaction being at the same date
	if !filter.Date.IsZero() {
		date := time.Date(filter.Date.Year(), filter.Date.Month(), filter.Date.Day(), 0, 0, 0, 0, time.UTC)
		query = query.Where("transactions.date >= date(?)", date).Where("transactions.date < date(?)", date.AddDate(0, 0, 1))
	}

	if !filter.FromDate.IsZero() {
		query = query.Where("transactions.date >= date(?)", time.Date(filter.FromDate.Year(), filter.FromDate.Month(), filter.FromDate.Day(), 0, 0, 0, 0, time.UTC))
	}

	if !filter.UntilDate.IsZero() {
		query = query.Where("transactions.date <= date(?)", time.Date(filter.UntilDate.Year(), filter.UntilDate.Month(), filter.UntilDate.Day(), 0, 0, 0, 0, time.UTC))
	}

	if filter.AccountID != "" {
		accountID, ok := httputil.UUIDFromString(c, filter.AccountID)
		if !ok {
			return
		}

		query = query.Where(co.DB.Where(&models.Transaction{
			TransactionCreate: models.TransactionCreate{
				SourceAccountID: accountID,
			},
		}).Or(&models.Transaction{
			TransactionCreate: models.TransactionCreate{
				DestinationAccountID: accountID,
			},
		}))
	}

	if !filter.AmountLessOrEqual.IsZero() {
		query = query.Where("transactions.amount <= ?", filter.AmountLessOrEqual)
	}

	if !filter.AmountMoreOrEqual.IsZero() {
		query = query.Where("transactions.amount >= ?", filter.AmountMoreOrEqual)
	}

	if filter.Note != "" {
		query = query.Where("note LIKE ?", fmt.Sprintf("%%%s%%", filter.Note))
	} else if slices.Contains(setFields, "Note") {
		query = query.Where("note = ''")
	}

	var transactions []models.Transaction
	if !queryWithRetry(c, query.Find(&transactions)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	if len(transactions) == 0 {
		transactions = make([]models.Transaction, 0)
	}

	c.JSON(http.StatusOK, TransactionListResponse{Data: transactions})
}

// GetTransaction returns a specific transaction
//
//	@Summary		Get transaction
//	@Description	Returns a specific transaction
//	@Tags			Transactions
//	@Produce		json
//	@Success		200	{object}	TransactionResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			transactionId	path		string	true	"ID formatted as string"
//	@Router			/v1/transactions/{transactionId} [get]
func (co Controller) GetTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var t models.Transaction
	err = co.DB.First(&t, id).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	c.JSON(http.StatusOK, TransactionResponse{Data: t})
}

// UpdateTransaction updates a specific transaction
//
//	@Summary		Update transaction
//	@Description	Updates an existing transaction. Only values to be updated need to be specified.
//	@Tags			Transactions
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	TransactionResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			transactionId	path		string						true	"ID formatted as string"
//	@Param			transaction		body		models.TransactionCreate	true	"Transaction"
//	@Router			/v1/transactions/{transactionId} [patch]
func (co Controller) UpdateTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	transaction, ok := co.getTransactionResource(c, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.TransactionCreate{})
	if err != nil {
		return
	}

	var data models.Transaction
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	// If the amount set via the API request is not existent or
	// is 0, we use the old amount
	if data.Amount.IsZero() {
		data.Amount = transaction.Amount
	}

	// Check the source account
	sourceAccountID := transaction.SourceAccountID
	if data.SourceAccountID != uuid.Nil {
		sourceAccountID = data.SourceAccountID
	}
	sourceAccount, ok := getResourceByID[models.Account](c, co, sourceAccountID)

	if !ok {
		return
	}

	// Check the destination account
	destinationAccountID := transaction.DestinationAccountID
	if data.DestinationAccountID != uuid.Nil {
		destinationAccountID = data.DestinationAccountID
	}
	destinationAccount, ok := getResourceByID[models.Account](c, co, destinationAccountID)

	if !ok {
		return
	}

	// Check the transaction that is set
	if !co.checkTransaction(c, data, sourceAccount, destinationAccount) {
		return
	}

	if !queryWithRetry(c, co.DB.Model(&transaction).Select("", updateFields...).Updates(data)) {
		return
	}

	// TODO: Remove
	// err = co.DB.First(&transaction, id).Error
	// if err != nil {
	// 	httperrors.Handler(c, err)
	// 	return
	// }
	c.JSON(http.StatusOK, TransactionResponse{Data: transaction})
}

// DeleteTransaction deletes a specific transaction
//
//	@Summary		Delete transaction
//	@Description	Deletes a transaction
//	@Tags			Transactions
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			transactionId	path		string	true	"ID formatted as string"
//	@Router			/v1/transactions/{transactionId} [delete]
func (co Controller) DeleteTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	transaction, ok := co.getTransactionResource(c, id)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&transaction)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// getTransactionResource verifies that the request URI is valid for the transaction and returns it.
func (co Controller) getTransactionResource(c *gin.Context, id uuid.UUID) (models.Transaction, bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "no transaction ID specified")
		return models.Transaction{}, false
	}

	var transaction models.Transaction

	if !queryWithRetry(c, co.DB.First(&transaction, &models.Transaction{
		DefaultModel: models.DefaultModel{
			ID: id,
		},
	}), "No transaction found for the specified ID") {
		return models.Transaction{}, false
	}

	return transaction, true
}

// checkTransaction verifies that the transaction is correct
//
// It checks that
//   - the transaction is not between two external accounts
//   - if an envelope is set: the transaction is not between two on-budget accounts
//   - if an envelope is set: the envelope exists
//
// It returns true if the transaction is valid, false in all
// other cases.
func (co Controller) checkTransaction(c *gin.Context, transaction models.Transaction, source, destination models.Account) (ok bool) {
	// If we don't mark the transaction as invalid, it is okay
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
		_, ok = getResourceByID[models.Envelope](c, co, *transaction.EnvelopeID)
	}

	return
}
