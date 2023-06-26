package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v2/pkg/httputil"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/gin-gonic/gin"
)

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
			r = append(r, ResponseTransactionV2{Data: t})
		}
	}

	c.JSON(status, r)
}
