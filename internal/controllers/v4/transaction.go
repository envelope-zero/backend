package v4

import (
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/v7/internal/httputil"
	"github.com/envelope-zero/backend/v7/internal/models"
	ez_uuid "github.com/envelope-zero/backend/v7/internal/uuid"
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
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/transactions/{id} [options]
func OptionsTransactionDetail(c *gin.Context) {
	resourceOptionsDetail(c, models.Transaction{})
}

// @Summary		Get transaction
// @Description	Returns a specific transaction
// @Tags			Transactions
// @Produce		json
// @Success		200	{object}	TransactionResponse
// @Failure		400	{object}	TransactionResponse
// @Failure		404	{object}	TransactionResponse
// @Failure		500	{object}	TransactionResponse
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/transactions/{id} [get]
func GetTransaction(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	var transaction models.Transaction
	err = models.DB.First(&transaction, uri.ID).Error
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
// @Param			date					query	string					false	"Date of the transaction. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			fromDate				query	string					false	"Transactions at and after this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			untilDate				query	string					false	"Transactions before and at this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			availableFromDate		query	string					false	"Availability date of the transaction. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			availableFromFromDate	query	string					false	"Transactions available at and after this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			availableFromUntilDate	query	string					false	"Transactions available before and at this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
// @Param			amount					query	string					false	"Filter by amount"
// @Param			amountLessOrEqual		query	string					false	"Amount less than or equal to this"
// @Param			amountMoreOrEqual		query	string					false	"Amount more than or equal to this"
// @Param			note					query	string					false	"Filter by note"
// @Param			budget					query	string					false	"Filter by budget ID"
// @Param			account					query	string					false	"Filter by ID of associated account, regardeless of source or destination"
// @Param			source					query	string					false	"Filter by source account ID"
// @Param			destination				query	string					false	"Filter by destination account ID"
// @Param			direction				query	TransactionDirection	false	"Filter by direction of transaction"
// @Param			envelope				query	string					false	"Filter by envelope ID"
// @Param			reconciledSource		query	bool					false	"Reconcilication state in source account"
// @Param			reconciledDestination	query	bool					false	"Reconcilication state in destination account"
// @Param			offset					query	uint					false	"The offset of the first Transaction returned. Defaults to 0."
// @Param			limit					query	int						false	"Maximum number of Transactions to return. Defaults to 50."
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

	// Filter for the transaction being available at the same date
	if !filter.AvailableFromDate.IsZero() {
		date := time.Date(filter.AvailableFromDate.Year(), filter.AvailableFromDate.Month(), filter.AvailableFromDate.Day(), 0, 0, 0, 0, time.UTC)
		q = q.Where("transactions.available_from >= date(?)", date).Where("transactions.available_from < date(?)", date.AddDate(0, 0, 1))
	}

	if !filter.AvailableFromFromDate.IsZero() {
		q = q.Where("transactions.available_from >= date(?)", time.Date(filter.AvailableFromFromDate.Year(), filter.AvailableFromFromDate.Month(), filter.AvailableFromFromDate.Day(), 0, 0, 0, 0, time.UTC))
	}

	if !filter.AvailableFromUntilDate.IsZero() {
		q = q.Where("transactions.available_from < date(?)", time.Date(filter.AvailableFromUntilDate.Year(), filter.AvailableFromUntilDate.Month(), filter.AvailableFromUntilDate.Day()+1, 0, 0, 0, 0, time.UTC))
	}

	if filter.BudgetID != ez_uuid.Nil {
		// We join on the source account ID since all resources need to belong to the
		// same budget anyways
		q = q.
			Joins("JOIN accounts on accounts.id = transactions.source_account_id").
			Joins("JOIN budgets on budgets.id = accounts.budget_id").
			Where("budgets.id = ?", filter.BudgetID)
	}

	if filter.AccountID != ez_uuid.Nil {
		q = q.Where(models.DB.Where(&models.Transaction{
			SourceAccountID: filter.AccountID.UUID,
		}).Or(&models.Transaction{
			DestinationAccountID: filter.AccountID.UUID,
		}))
	}

	if filter.Direction != "" {
		if !slices.Contains([]TransactionDirection{DirectionIn, DirectionOut, DirectionInternal}, filter.Direction) {
			s := errTransactionDirectionInvalid.Error()
			c.JSON(http.StatusBadRequest, TransactionListResponse{
				Error: &s,
			})
			return
		}

		// Internal transactions are internal account to internal account
		if filter.Direction == DirectionInternal {
			q = q.
				Joins("JOIN accounts AS direction_accounts_source on direction_accounts_source.id = transactions.source_account_id").
				Joins("JOIN accounts AS direction_accounts_destination on direction_accounts_destination.id = transactions.destination_account_id").
				Where("direction_accounts_source.external = false AND direction_accounts_destination.external = false")
		}

		// Transactions going in are external account to internal account
		if filter.Direction == DirectionIn {
			q = q.
				Joins("JOIN accounts AS direction_accounts_source on direction_accounts_source.id = transactions.source_account_id").
				Joins("JOIN accounts AS direction_accounts_destination on direction_accounts_destination.id = transactions.destination_account_id").
				Where("direction_accounts_source.external = true AND direction_accounts_destination.external = false")
		}

		// Transactions going out are internal account to external account
		if filter.Direction == DirectionOut {
			q = q.
				Joins("JOIN accounts AS direction_accounts_source on direction_accounts_source.id = transactions.source_account_id").
				Joins("JOIN accounts AS direction_accounts_destination on direction_accounts_destination.id = transactions.destination_account_id").
				Where("direction_accounts_source.external = false AND direction_accounts_destination.external = true")
		}
	}

	if filter.Type != "" {
		if !slices.Contains([]TransactionType{TypeIncome, TypeSpend, TypeTransfer}, filter.Type) {
			s := errTransactionTypeInvalid.Error()
			c.JSON(http.StatusBadRequest, TransactionListResponse{
				Error: &s,
			})
			return
		}

		// Income is coming from an off-budget to an on-budget account
		if filter.Type == TypeIncome {
			q = q.
				Joins("JOIN accounts AS type_accounts_source on type_accounts_source.id = transactions.source_account_id").
				Joins("JOIN accounts AS type_accounts_destination on type_accounts_destination.id = transactions.destination_account_id").
				Where("type_accounts_source.on_budget = false AND type_accounts_destination.on_budget = true")
		}

		// Spend is going from an on-budget to an off-budget account
		if filter.Type == TypeSpend {
			q = q.
				Joins("JOIN accounts AS type_accounts_source on type_accounts_source.id = transactions.source_account_id").
				Joins("JOIN accounts AS type_accounts_destination on type_accounts_destination.id = transactions.destination_account_id").
				Where("type_accounts_source.on_budget = true AND type_accounts_destination.on_budget = false")
		}

		// Transfers are going from an on-budget to an on-budget account
		if filter.Type == TypeTransfer {
			q = q.
				Joins("JOIN accounts AS type_accounts_source on type_accounts_source.id = transactions.source_account_id").
				Joins("JOIN accounts AS type_accounts_destination on type_accounts_destination.id = transactions.destination_account_id").
				Where("type_accounts_source.on_budget = true AND type_accounts_destination.on_budget = true")
		}
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
// @Param			id			path		URIID				true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			transaction	body		TransactionEditable	true	"Transaction"
// @Router			/v4/transactions/{id} [patch]
func UpdateTransaction(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), TransactionResponse{
			Error: &e,
		})
		return
	}

	// Get the transaction resource
	var transaction models.Transaction
	err = models.DB.First(&transaction, uri.ID).Error
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
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/transactions/{id} [delete]
func DeleteTransaction(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var transaction models.Transaction
	err = models.DB.First(&transaction, uri.ID).Error
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
