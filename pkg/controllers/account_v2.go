package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountV2 is the API v2 representation of an Account in EZ.
type AccountV2 struct {
	models.Account
	Balance           decimal.Decimal `json:"balance" example:"2735.17"`           // Balance of the account, including all transactions referencing it
	ReconciledBalance decimal.Decimal `json:"reconciledBalance" example:"2539.57"` // Balance of the account, including all reconciled transactions referencing it
	RecentEnvelopes   []*uuid.UUID    `json:"recentEnvelopes"`                     // Envelopes recently used with this account

	Links struct {
		Self         string `json:"self" example:"https://example.com/api/v2/accounts/af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`                     // The account itself
		Transactions string `json:"transactions" example:"https://example.com/api/v2/transactions?account=af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"` // Transactions referencing the account
	} `json:"links"`
}

// links generates the HATEOAS links for the Account.
func (a *AccountV2) links(c *gin.Context) {
	url := c.GetString(string(database.ContextURL))
	a.Links.Self = fmt.Sprintf("%s/v2/accounts/%s", url, a.ID)
	a.Links.Transactions = fmt.Sprintf("%s/v2/transactions?account=%s", url, a.ID)
}

func (co Controller) getAccountV2(c *gin.Context, id uuid.UUID) (AccountV2, bool) {
	accountModel, ok := getResourceByIDAndHandleErrors[models.Account](c, co, id)

	account := AccountV2{
		Account: accountModel,
	}

	if !ok {
		return AccountV2{}, false
	}

	// Recent Envelopes
	ids, err := accountModel.RecentEnvelopes(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return AccountV2{}, false
	}

	account.RecentEnvelopes = ids

	// Balance
	balance, _, err := accountModel.GetBalanceMonth(co.DB, types.Month{})
	if err != nil {
		httperrors.Handler(c, err)
		return AccountV2{}, false
	}
	account.Balance = balance

	// Reconciled Balance
	reconciledBalance, err := accountModel.SumReconciled(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return AccountV2{}, false
	}
	account.ReconciledBalance = reconciledBalance

	// Links
	account.links(c)

	return account, true
}

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
//	@Router			/v2/accounts [options]
//	@Deprecated		true
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
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v2/accounts [get]
//	@Param			name		query	string	false	"Filter by name"
//	@Param			note		query	string	false	"Filter by note"
//	@Param			budget		query	string	false	"Filter by budget ID"
//	@Param			onBudget	query	bool	false	"Is the account on-budget?"
//	@Param			external	query	bool	false	"Is the account external?"
//	@Param			hidden		query	bool	false	"Is the account hidden?"
//	@Param			search		query	string	false	"Search for this text in name and note"
//	@Deprecated		true
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

	query := co.DB.Where(&models.Account{
		AccountCreate: create,
	}, queryFields...)

	query = stringFilters(co.DB, query, setFields, filter.Name, filter.Note, filter.Search)

	var accounts []models.Account
	if !queryAndHandleErrors(c, query.Find(&accounts)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	accountObjects := make([]AccountV2, 0)

	for _, account := range accounts {
		o, _ := co.getAccountV2(c, account.ID)
		accountObjects = append(accountObjects, o)
	}

	c.JSON(http.StatusOK, accountObjects)
}
