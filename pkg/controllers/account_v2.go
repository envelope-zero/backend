package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RegisterAccountRoutes registers the routes for accounts with
// the RouterGroup that is passed.
func (co Controller) RegisterAccountRoutesV2(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsAccountListV2)
		r.GET("", co.GetAccountsV2)
	}
}

// OptionsAccountList returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Accounts
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Router			/v2/accounts [options]
func (co Controller) OptionsAccountListV2(c *gin.Context) {
	httputil.OptionsGet(c)
}

// GetAccounts returns a list of all accounts matching the filter parameters
//
//	@Summary		List accounts
//	@Description	Returns a list of accounts
//	@Tags			Accounts
//	@Produce		json
//	@Success		200	{object}	AccountListResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v2/accounts [get]
//	@Param			name		query	string	false	"Filter by name"
//	@Param			note		query	string	false	"Filter by note"
//	@Param			budget		query	string	false	"Filter by budget ID"
//	@Param			onBudget	query	bool	false	"Is the account on-budget?"
//	@Param			external	query	bool	false	"Is the account external?"
//	@Param			hidden		query	bool	false	"Is the account hidden?"
//	@Param			search		query	string	false	"Search for this text in name and note"
func (co Controller) GetAccountsV2(c *gin.Context) {
	var filter AccountQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the set parameters in the query string
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, ok := filter.ToCreate(c)
	if !ok {
		return
	}

	query := co.DB.Where(&models.AccountV2{
		AccountCreate: create,
	}, queryFields...)

	query = stringFilters(co.DB, query, setFields, filter.Name, filter.Note, filter.Search)

	var accounts []models.AccountV2
	if !queryWithRetry(c, query.Find(&accounts)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	accountObjects := make([]models.AccountV2, 0)

	for _, account := range accounts {
		o, _ := co.getAccountObjectV2(c, account.ID)
		accountObjects = append(accountObjects, o)
	}

	c.JSON(http.StatusOK, accountObjects)
}

// getAccountObjectV2 returns the account object with all calculations done.
func (co Controller) getAccountObjectV2(c *gin.Context, id uuid.UUID) (models.AccountV2, bool) {
	account, ok := getResourceByIDAndHandleErrors[models.AccountV2](c, co, id)

	if !ok {
		return models.AccountV2{}, false
	}

	err := account.SetRecentEnvelopes(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return models.AccountV2{}, false
	}

	err = account.WithCalculations(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return models.AccountV2{}, false
	}

	return account, true
}
