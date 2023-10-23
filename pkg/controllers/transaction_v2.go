package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TransactionV2 Transaction

// links generates HATEOAS links for the transaction.
func (t *TransactionV2) links(c *gin.Context) {
	// Set links
	t.Links.Self = fmt.Sprintf("%s/v2/transactions/%s", c.GetString(string(database.ContextURL)), t.ID)
}

func (co Controller) getTransactionV2(c *gin.Context, id uuid.UUID) (TransactionV2, bool) {
	transactionModel, ok := getResourceByIDAndHandleErrors[models.Transaction](c, co, id)
	if !ok {
		return TransactionV2{}, false
	}

	transaction := TransactionV2{
		Transaction: transactionModel,
	}

	transaction.links(c)
	return transaction, true
}

// RegisterTransactionRoutesV2 registers the routes for transactions with
// the RouterGroup that is passed.
func (co Controller) RegisterTransactionRoutesV2(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsTransactionsV2)
		r.POST("", co.CreateTransactionsV2)
	}
}

// OptionsTransactionsV2 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Transactions
//	@Success		204
//	@Router			/v2/transactions [options]
func (co Controller) OptionsTransactionsV2(c *gin.Context) {
	httputil.OptionsPost(c)
}

// CreateTransactionsV2 creates transactions
//
//	@Summary		Create transactions
//	@Description	Creates transactions from the list of submitted transaction data. The response code is the highest response code number that a single transaction creation would have caused. If it is not equal to 201, at least one transaction has an error.
//	@Tags			Transactions
//	@Produce		json
//	@Success		201	{object}	[]ResponseTransactionV2
//	@Failure		400	{object}	[]ResponseTransactionV2
//	@Failure		404
//	@Failure		500				{object}	[]ResponseTransactionV2
//	@Param			transactions	body		[]models.TransactionCreate	true	"Transactions"
//	@Router			/v2/transactions [post]
func (co Controller) CreateTransactionsV2(c *gin.Context) {
	var transactions []models.Transaction

	if err := httputil.BindData(c, &transactions); err != nil {
		return
	}

	// The response list has the same length as the request list
	r := make([]ResponseTransactionV2, 0, len(transactions))

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated

	for _, t := range transactions {
		t, err := co.createTransaction(c, t)

		// Append the error or the successfully created transaction to the response list
		if !err.Nil() {
			r = append(r, ResponseTransactionV2{Error: err.Error()})

			// The final status code is the highest HTTP status code number since this also
			// represents the priority we
			if err.Status > status {
				status = err.Status
			}
		} else {
			tObject, ok := co.getTransactionV2(c, t.ID)
			if !ok {
				return
			}
			r = append(r, ResponseTransactionV2{Data: tObject})
		}
	}

	c.JSON(status, r)
}
