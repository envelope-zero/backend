package v4

import (
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

// RegisterTransactionRoutes registers the routes for transactions with
// the RouterGroup that is passed.
func RegisterTransactionRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsTransactions)
		r.GET("", GetTransactions)
		r.POST("", CreateTransactions)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:id", OptionsTransactionDetail)
		r.GET("/:id", GetTransaction)
		r.PATCH("/:id", UpdateTransaction)
		r.DELETE("/:id", DeleteTransaction)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Transactions
// @Success		204
// @Router			/v4/transactions [options]
func OptionsTransactions(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Transactions
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/transactions/{id} [options]
func OptionsTransactionDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var t models.Transaction
	err = models.DB.First(&t, id).Error
	if err != nil {
		c.JSON(status(err), httpError{
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
// @Success		200	{object}	TransactionResponse
// @Failure		400	{object}	TransactionResponse
// @Failure		404	{object}	TransactionResponse
// @Failure		500	{object}	TransactionResponse
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/transactions/{id} [get]
func GetTransaction(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	var transaction models.Transaction
	err = models.DB.First(&transaction, id).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	data := newTransaction(c, transaction)
	c.JSON(http.StatusOK, TransactionResponse{Data: &data})
}

// @Summary		Get transactions
// @Description	Returns a list of transactions
// @Tags			Transactions
// @Produce		json
// @Success		200	{object}	TransactionListResponse
// @Failure		400	{object}	TransactionListResponse
// @Failure		500	{object}	TransactionListResponse
// @Router			/v4/transactions [get]
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
func GetTransactions(c *gin.Context) {
	var filter TransactionQueryFilter
	if err := c.Bind(&filter); err != nil {
		s := err.Error()
		c.JSON(http.StatusBadRequest, TransactionListResponse{
			Error: &s,
		})
		return
	}

	// Get the fields set in the filter
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	model, err := filter.model()
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionListResponse{
			Error: &e,
		})
		return
	}

	var q *gorm.DB
	q = models.DB.Order("datetime(transactions.date) DESC, datetime(transactions.created_at) DESC").Where(&model, queryFields...)

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

	if filter.BudgetID != "" {
		budgetID, err := httputil.UUIDFromString(filter.BudgetID)
		if err != nil {
			s := fmt.Sprintf("Error parsing budget ID for filtering: %s", err.Error())
			c.JSON(status(err), TransactionListResponse{
				Error: &s,
			})
			return
		}

		// We join on the source account ID since all resources need to belong to the
		// same budget anyways
		q = q.
			Joins("JOIN accounts on accounts.id = transactions.source_account_id ").
			Joins("JOIN budgets on budgets.id = accounts.budget_id").
			Where("budgets.id = ?", budgetID)
	}

	if filter.AccountID != "" {
		accountID, err := httputil.UUIDFromString(filter.AccountID)
		if err != nil {
			s := fmt.Sprintf("Error parsing Account ID for filtering: %s", err.Error())
			c.JSON(status(err), TransactionListResponse{
				Error: &s,
			})
			return
		}

		q = q.Where(models.DB.Where(&models.Transaction{
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
		q = q.Where("transactions.note LIKE ?", fmt.Sprintf("%%%s%%", filter.Note))
	} else if slices.Contains(setFields, "Note") {
		q = q.Where("transactions.note = ''")
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
	err = q.Find(&transactions).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionListResponse{
			Error: &e,
		})
		return
	}

	var count int64
	err = q.Limit(-1).Offset(-1).Count(&count).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionListResponse{
			Error: &e,
		})
		return
	}

	data := make([]Transaction, 0)
	for _, transaction := range transactions {
		data = append(data, newTransaction(c, transaction))
	}

	c.JSON(http.StatusOK, TransactionListResponse{
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
// @Success		201				{object}	TransactionCreateResponse
// @Failure		400				{object}	TransactionCreateResponse
// @Failure		404				{object}	TransactionCreateResponse
// @Failure		500				{object}	TransactionCreateResponse
// @Param			transactions	body		[]TransactionEditable	true	"Transactions"
// @Router			/v4/transactions [post]
func CreateTransactions(c *gin.Context) {
	var editables []TransactionEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &editables)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := TransactionCreateResponse{}

	for _, editable := range editables {
		transaction := editable.model()
		err := models.DB.Create(&transaction).Error
		// Append the error
		if err != nil {
			status = r.appendError(err, status)
			continue
		}

		data := newTransaction(c, transaction)
		r.Data = append(r.Data, TransactionResponse{Data: &data})
	}

	c.JSON(status, r)
}

// @Summary		Update transaction
// @Description	Updates an existing transaction. Only values to be updated need to be specified.
// @Tags			Transactions
// @Accept			json
// @Produce		json
// @Success		200			{object}	TransactionResponse
// @Failure		400			{object}	TransactionResponse
// @Failure		404			{object}	TransactionResponse
// @Failure		500			{object}	TransactionResponse
// @Param			id			path		string				true	"ID formatted as string"
// @Param			transaction	body		TransactionEditable	true	"Transaction"
// @Router			/v4/transactions/{id} [patch]
func UpdateTransaction(c *gin.Context) {
	// Get the resource ID
	id, err := httputil.UUIDFromString(c.Param("id"))
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	// Get the transaction resource
	var transaction models.Transaction
	err = models.DB.First(&transaction, id).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	// Get the fields that are set to be updated
	updateFields, err := httputil.GetBodyFields(c, TransactionEditable{})
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	// Bind the update for the patch
	var update TransactionEditable
	err = httputil.BindData(c, &update)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	// If the amount set via the API request is not existent or
	// is 0, we use the old amount
	if update.Amount.IsZero() {
		update.Amount = transaction.Amount
	}

	err = models.DB.Model(&transaction).Select("", updateFields...).Updates(update.model()).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	data := newTransaction(c, transaction)
	c.JSON(http.StatusOK, TransactionResponse{Data: &data})
}

// @Summary		Delete transaction
// @Description	Deletes a transaction
// @Tags			Transactions
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/transactions/{id} [delete]
func DeleteTransaction(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var transaction models.Transaction
	err = models.DB.First(&transaction, id).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.Delete(&transaction).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
