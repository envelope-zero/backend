package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

type TransactionListResponseV3 struct {
	Data       []TransactionV3 `json:"data"`                                                          // List of transactions
	Error      *string         `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination     `json:"pagination"`                                                    // Pagination information
}

type TransactionCreateResponseV3 struct {
	Error *string                 `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []TransactionResponseV3 `json:"data"`                                                          // List of created Transactions
}

type TransactionResponseV3 struct {
	Error *string        `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this transaction
	Data  *TransactionV3 `json:"data"`                                                          // The Transaction data, if creation was successful
}

// TransactionV3 is the representation of a Transaction in API v3.
type TransactionV3 struct {
	models.Transaction
	Reconciled bool `json:"reconciled,omitempty"` // Remove the reconciled field
	Links      struct {
		Self string `json:"self" example:"https://example.com/api/v3/transactions/d430d7c3-d14c-4712-9336-ee56965a6673"` // The transaction itself
	} `json:"links"` // Links for the transaction
}

// links generates HATEOAS links for the transaction.
func (t *TransactionV3) links(c *gin.Context) {
	// Set links
	t.Links.Self = fmt.Sprintf("%s/v3/transactions/%s", c.GetString(string(database.ContextURL)), t.ID)
}

type TransactionQueryFilterV3 struct {
	Date                  time.Time       `form:"date" filterField:"false"`              // Exact date. Time is ignored.
	FromDate              time.Time       `form:"fromDate" filterField:"false"`          // From this date. Time is ignored.
	UntilDate             time.Time       `form:"untilDate" filterField:"false"`         // Until this date. Time is ignored.
	Amount                decimal.Decimal `form:"amount"`                                // Exact amount
	AmountLessOrEqual     decimal.Decimal `form:"amountLessOrEqual" filterField:"false"` // Amount less than or equal to this
	AmountMoreOrEqual     decimal.Decimal `form:"amountMoreOrEqual" filterField:"false"` // Amount more than or equal to this
	Note                  string          `form:"note" filterField:"false"`              // Note contains this string
	BudgetID              string          `form:"budget"`                                // ID of the budget
	SourceAccountID       string          `form:"source"`                                // ID of the source account
	DestinationAccountID  string          `form:"destination"`                           // ID of the destination account
	EnvelopeID            string          `form:"envelope"`                              // ID of the envelope
	ReconciledSource      bool            `form:"reconciledSource"`                      // Is the transaction reconciled in the source account?
	ReconciledDestination bool            `form:"reconciledDestination"`                 // Is the transaction reconciled in the destination account?
	AccountID             string          `form:"account" filterField:"false"`           // ID of either source or destination account
	Offset                uint            `form:"offset" filterField:"false"`            // The offset of the first Transaction returned. Defaults to 0.
	Limit                 int             `form:"limit" filterField:"false"`             // Maximum number of transactions to return. Defaults to 50.
}

func (co Controller) getTransactionV3(c *gin.Context, id uuid.UUID) (TransactionV3, httperrors.Error) {
	transactionModel, err := getResourceByID[models.Transaction](c, co, id)
	if !err.Nil() {
		return TransactionV3{}, err
	}

	transaction := TransactionV3{
		Transaction: transactionModel,
	}

	transaction.links(c)
	return transaction, httperrors.Error{}
}

// RegisterTransactionRoutesV3 registers the routes for transactions with
// the RouterGroup that is passed.
func (co Controller) RegisterTransactionRoutesV3(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsTransactionsV3)
		r.GET("", co.GetTransactionsV3)
		r.POST("", co.CreateTransactionsV3)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:id", co.OptionsTransactionDetailV3)
		r.GET("/:id", co.GetTransactionV3)
		r.PATCH("/:id", co.UpdateTransactionV3)
		r.DELETE("/:id", co.DeleteTransactionV3)
	}
}

// OptionsTransactionsV3 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Transactions
//	@Success		204
//	@Router			/v3/transactions [options]
func (co Controller) OptionsTransactionsV3(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsTransactionDetailV3 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Transactions
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v3/transactions/{id} [options]
func (co Controller) OptionsTransactionDetailV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	var t models.Transaction
	err = query(c, co.DB.First(&t, id))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// GetTransactionV3 returns a specific transaction
//
//	@Summary		Get transaction
//	@Description	Returns a specific transaction
//	@Tags			Transactions
//	@Produce		json
//	@Success		200	{object}	TransactionResponseV3
//	@Failure		400	{object}	TransactionResponseV3
//	@Failure		404	{object}	TransactionResponseV3
//	@Failure		500	{object}	TransactionResponseV3
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v3/transactions/{id} [get]
func (co Controller) GetTransactionV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	var t models.Transaction
	err = query(c, co.DB.First(&t, id))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	tObject, err := co.getTransactionV3(c, t.ID)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	c.JSON(http.StatusOK, TransactionResponseV3{Data: &tObject})
}

// GetTransactions returns transactions filtered by the query parameters
//
//	@Summary		Get transactions
//	@Description	Returns a list of transactions
//	@Tags			Transactions
//	@Produce		json
//	@Success		200	{object}	TransactionListResponseV3
//	@Failure		400	{object}	TransactionListResponseV3
//	@Failure		500	{object}	TransactionListResponseV3
//	@Router			/v3/transactions [get]
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
//	@Param			reconciledSource		query	bool	false	"Reconcilication state in source account"
//	@Param			reconciledDestination	query	bool	false	"Reconcilication state in destination account"
//	@Param			offset					query	uint	false	"The offset of the first Transaction returned. Defaults to 0."
//	@Param			limit					query	int		false	"Maximum number of transactions to return. Defaults to 50."
func (co Controller) GetTransactionsV3(c *gin.Context) {
	var filter TransactionQueryFilterV3
	if err := c.Bind(&filter); err != nil {
		s := httperrors.ErrInvalidQueryString.Error()
		c.JSON(http.StatusBadRequest, TransactionListResponseV3{
			Error: &s,
		})
		return
	}

	// Get the fields set in the filter
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate()
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionListResponseV3{
			Error: &e,
		})
		return
	}

	var q *gorm.DB
	q = co.DB.Order("datetime(date) DESC, datetime(created_at) DESC").Where(&models.Transaction{
		TransactionCreate: create,
	}, queryFields...)

	// Filter for the transaction being at the same date
	if !filter.Date.IsZero() {
		date := time.Date(filter.Date.Year(), filter.Date.Month(), filter.Date.Day(), 0, 0, 0, 0, time.UTC)
		q = q.Where("transactions.date >= date(?)", date).Where("transactions.date < date(?)", date.AddDate(0, 0, 1))
	}

	if !filter.FromDate.IsZero() {
		q = q.Where("transactions.date >= date(?)", time.Date(filter.FromDate.Year(), filter.FromDate.Month(), filter.FromDate.Day(), 0, 0, 0, 0, time.UTC))
	}

	if !filter.UntilDate.IsZero() {
		q = q.Where("transactions.date < date(?)", time.Date(filter.UntilDate.Year(), filter.UntilDate.Month(), filter.UntilDate.Day()+1, 0, 0, 0, 0, time.UTC))
	}

	if filter.AccountID != "" {
		accountID, err := httputil.UUIDFromString(filter.AccountID)
		if !err.Nil() {
			s := fmt.Sprintf("Error parsing Account ID for filtering: %s", err.Error())
			c.JSON(err.Status, TransactionListResponseV3{
				Error: &s,
			})
			return
		}

		q = q.Where(co.DB.Where(&models.Transaction{
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
		q = q.Where("transactions.amount <= ?", filter.AmountLessOrEqual)
	}

	if !filter.AmountMoreOrEqual.IsZero() {
		q = q.Where("transactions.amount >= ?", filter.AmountMoreOrEqual)
	}

	if filter.Note != "" {
		q = q.Where("note LIKE 	?", fmt.Sprintf("%%%s%%", filter.Note))
	} else if slices.Contains(setFields, "Note") {
		q = q.Where("note = ''")
	}

	// Set the offset. Does not need checking since the default is 0
	q = q.Offset(int(filter.Offset))

	// Default to 50 transactions and set the limit
	limit := 50
	if slices.Contains(setFields, "Limit") {
		limit = filter.Limit
	}
	q = q.Limit(limit)

	var transactions []models.Transaction
	err = query(c, q.Find(&transactions))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionListResponseV3{
			Error: &e,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionListResponseV3{
			Error: &e,
		})
		return
	}

	transactionObjects := make([]TransactionV3, 0)
	for _, t := range transactions {
		transactionObject, err := co.getTransactionV3(c, t.ID)
		if !err.Nil() {
			e := err.Error()
			c.JSON(err.Status, TransactionListResponseV3{
				Error: &e,
			})
			return
		}

		transactionObjects = append(transactionObjects, transactionObject)
	}

	c.JSON(http.StatusOK, TransactionListResponseV3{
		Data: transactionObjects,
		Pagination: &Pagination{
			Count:  len(transactionObjects),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// CreateTransactionsV3 creates transactions
//
//	@Summary		Create transactions
//	@Description	Creates transactions from the list of submitted transaction data. The response code is the highest response code number that a single transaction creation would have caused. If it is not equal to 201, at least one transaction has an error.
//	@Tags			Transactions
//	@Produce		json
//	@Success		201				{object}	TransactionCreateResponseV3
//	@Failure		400				{object}	TransactionCreateResponseV3
//	@Failure		404				{object}	TransactionCreateResponseV3
//	@Failure		500				{object}	TransactionCreateResponseV3
//	@Param			transactions	body		[]models.TransactionCreate	true	"Transactions"
//	@Router			/v3/transactions [post]
func (co Controller) CreateTransactionsV3(c *gin.Context) {
	var transactions []models.TransactionCreate

	// Bind data and return error if not possible
	err := httputil.BindData(c, &transactions)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionCreateResponseV3{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := TransactionCreateResponseV3{}

	for _, create := range transactions {
		t, err := co.createTransaction(c, create)

		// Append the error
		if !err.Nil() {
			e := err.Error()
			r.Data = append(r.Data, TransactionResponseV3{Error: &e})

			// The final status code is the highest HTTP status code number since this also
			// represents the priority we
			if err.Status > status {
				status = err.Status
			}
			continue
		}

		// Append the transaction
		tObject, err := co.getTransactionV3(c, t.ID)
		if !err.Nil() {
			e := err.Error()
			c.JSON(err.Status, TransactionCreateResponseV3{
				Error: &e,
			})
			return
		}
		r.Data = append(r.Data, TransactionResponseV3{Data: &tObject})
	}

	c.JSON(status, r)
}

// UpdateTransactionV3 updates a specific transaction
//
//	@Summary		Update transaction
//	@Description	Updates an existing transaction. Only values to be updated need to be specified.
//	@Tags			Transactions
//	@Accept			json
//	@Produce		json
//	@Success		200			{object}	TransactionResponseV3
//	@Failure		400			{object}	TransactionResponseV3
//	@Failure		404			{object}	TransactionResponseV3
//	@Failure		500			{object}	TransactionResponseV3
//	@Param			id			path		string						true	"ID formatted as string"
//	@Param			transaction	body		models.TransactionCreate	true	"Transaction"
//	@Router			/v3/transactions/{id} [patch]
func (co Controller) UpdateTransactionV3(c *gin.Context) {
	// Get the resource ID
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	// Get the transaction resource
	transaction, err := getResourceByID[models.Transaction](c, co, id)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	// Get the fields that are set to be updated
	updateFields, err := httputil.GetBodyFields(c, models.TransactionCreate{})
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	// Bind the data for the patch
	var data models.Transaction
	err = httputil.BindData(c, &data.TransactionCreate)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
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
	sourceAccount, err := getResourceByID[models.Account](c, co, sourceAccountID)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	// Check the destination account
	destinationAccountID := transaction.DestinationAccountID
	if data.DestinationAccountID != uuid.Nil {
		destinationAccountID = data.DestinationAccountID
	}
	destinationAccount, err := getResourceByID[models.Account](c, co, destinationAccountID)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	// Check the transaction that is set
	err = co.checkTransaction(c, data, sourceAccount, destinationAccount)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	err = query(c, co.DB.Model(&transaction).Select("", updateFields...).Updates(data))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	transactionObject, err := co.getTransactionV3(c, transaction.ID)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	c.JSON(http.StatusOK, TransactionResponseV3{Data: &transactionObject})
}

// DeleteTransactionV3 deletes a specific transaction
//
//	@Summary		Delete transaction
//	@Description	Deletes a transaction
//	@Tags			Transactions
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v3/transactions/{id} [delete]
func (co Controller) DeleteTransactionV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	transaction, err := getResourceByID[models.Transaction](c, co, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, co.DB.Delete(&transaction))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
