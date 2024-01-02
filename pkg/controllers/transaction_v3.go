package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

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

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Transactions
// @Success		204
// @Router			/v3/transactions [options]
func (co Controller) OptionsTransactionsV3(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Transactions
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/transactions/{id} [options]
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

// @Summary		Get transaction
// @Description	Returns a specific transaction
// @Tags			Transactions
// @Produce		json
// @Success		200	{object}	TransactionResponseV3
// @Failure		400	{object}	TransactionResponseV3
// @Failure		404	{object}	TransactionResponseV3
// @Failure		500	{object}	TransactionResponseV3
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/transactions/{id} [get]
func (co Controller) GetTransactionV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	var transaction models.Transaction
	err = query(c, co.DB.First(&transaction, id))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	data := newTransactionV3(c, transaction)
	c.JSON(http.StatusOK, TransactionResponseV3{Data: &data})
}

// @Summary		Get transactions
// @Description	Returns a list of transactions
// @Tags			Transactions
// @Produce		json
// @Success		200	{object}	TransactionListResponseV3
// @Failure		400	{object}	TransactionListResponseV3
// @Failure		500	{object}	TransactionListResponseV3
// @Router			/v3/transactions [get]
// @Param			date					query	string	false	"Date of the transaction. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			fromDate				query	string	false	"Transactions at and after this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			untilDate				query	string	false	"Transactions before and at this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			amount					query	string	false	"Filter by amount"
// @Param			amountLessOrEqual		query	string	false	"Amount less than or equal to this"
// @Param			amountMoreOrEqual		query	string	false	"Amount more than or equal to this"
// @Param			note					query	string	false	"Filter by note"
// @Param			budget					query	string	false	"Filter by budget ID"
// @Param			account					query	string	false	"Filter by ID of associated account, regardeless of source or destination"
// @Param			source					query	string	false	"Filter by source account ID"
// @Param			destination				query	string	false	"Filter by destination account ID"
// @Param			envelope				query	string	false	"Filter by envelope ID"
// @Param			reconciledSource		query	bool	false	"Reconcilication state in source account"
// @Param			reconciledDestination	query	bool	false	"Reconcilication state in destination account"
// @Param			offset					query	uint	false	"The offset of the first Transaction returned. Defaults to 0."
// @Param			limit					query	int		false	"Maximum number of Transactions to return. Defaults to 50."
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
	model, err := filter.model()
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionListResponseV3{
			Error: &e,
		})
		return
	}

	var q *gorm.DB
	q = co.DB.Order("datetime(date) DESC, datetime(created_at) DESC").Where(&model, queryFields...)

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
			SourceAccountID: accountID,
		}).Or(&models.Transaction{
			DestinationAccountID: accountID,
		}))
	}

	if !filter.AmountLessOrEqual.IsZero() {
		q = q.Where("transactions.amount <= ?", filter.AmountLessOrEqual)
	}

	if !filter.AmountMoreOrEqual.IsZero() {
		q = q.Where("transactions.amount >= ?", filter.AmountMoreOrEqual)
	}

	if filter.Note != "" {
		q = q.Where("note LIKE ?", fmt.Sprintf("%%%s%%", filter.Note))
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

	data := make([]TransactionV3, 0)
	for _, transaction := range transactions {
		data = append(data, newTransactionV3(c, transaction))
	}

	c.JSON(http.StatusOK, TransactionListResponseV3{
		Data: data,
		Pagination: &Pagination{
			Count:  len(data),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// @Summary		Create transactions
// @Description	Creates transactions from the list of submitted transaction data. The response code is the highest response code number that a single transaction creation would have caused. If it is not equal to 201, at least one transaction has an error.
// @Tags			Transactions
// @Produce		json
// @Success		201				{object}	TransactionCreateResponseV3
// @Failure		400				{object}	TransactionCreateResponseV3
// @Failure		404				{object}	TransactionCreateResponseV3
// @Failure		500				{object}	TransactionCreateResponseV3
// @Param			transactions	body		[]TransactionV3Editable	true	"Transactions"
// @Router			/v3/transactions [post]
func (co Controller) CreateTransactionsV3(c *gin.Context) {
	var editables []TransactionV3Editable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &editables)
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

	for _, editable := range editables {
		transaction := editable.model()

		err := co.createTransaction(c, &transaction)

		// Append the error
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		data := newTransactionV3(c, transaction)
		r.Data = append(r.Data, TransactionResponseV3{Data: &data})
	}

	c.JSON(status, r)
}

// @Summary		Update transaction
// @Description	Updates an existing transaction. Only values to be updated need to be specified.
// @Tags			Transactions
// @Accept			json
// @Produce		json
// @Success		200			{object}	TransactionResponseV3
// @Failure		400			{object}	TransactionResponseV3
// @Failure		404			{object}	TransactionResponseV3
// @Failure		500			{object}	TransactionResponseV3
// @Param			id			path		string					true	"ID formatted as string"
// @Param			transaction	body		TransactionV3Editable	true	"Transaction"
// @Router			/v3/transactions/{id} [patch]
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
	updateFields, err := httputil.GetBodyFields(c, TransactionV3Editable{})
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	// Bind the update for the patch
	var update TransactionV3Editable
	err = httputil.BindData(c, &update)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	// If the amount set via the API request is not existent or
	// is 0, we use the old amount
	if update.Amount.IsZero() {
		update.Amount = transaction.Amount
	}

	// Check the source account
	sourceAccountID := transaction.SourceAccountID
	if update.SourceAccountID != uuid.Nil {
		sourceAccountID = update.SourceAccountID
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
	if update.DestinationAccountID != uuid.Nil {
		destinationAccountID = update.DestinationAccountID
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
	err = co.checkTransaction(c, update.model(), sourceAccount, destinationAccount)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	err = query(c, co.DB.Model(&transaction).Select("", updateFields...).Updates(update.model()))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionResponseV3{
			Error: &e,
		})
		return
	}

	data := newTransactionV3(c, transaction)
	c.JSON(http.StatusOK, TransactionResponseV3{Data: &data})
}

// @Summary		Delete transaction
// @Description	Deletes a transaction
// @Tags			Transactions
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/transactions/{id} [delete]
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
