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

// Account is the API v1 representation of an Account in EZ.
type Account struct {
	models.Account
	Balance           decimal.Decimal   `json:"balance" example:"2735.17"`           // Balance of the account, including all transactions referencing it
	ReconciledBalance decimal.Decimal   `json:"reconciledBalance" example:"2539.57"` // Balance of the account, including all reconciled transactions referencing it
	RecentEnvelopes   []models.Envelope `json:"recentEnvelopes"`                     // Envelopes recently used with this account

	Links struct {
		Self         string `json:"self" example:"https://example.com/api/v1/accounts/af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`                     // The account itself
		Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions?account=af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"` // Transactions referencing the account
	} `json:"links"`
}

// links generates the HATEOAS links for the Account.
func (a *Account) links(c *gin.Context) {
	url := c.GetString(string(database.ContextURL))
	a.Links.Self = fmt.Sprintf("%s/v1/accounts/%s", url, a.ID)
	a.Links.Transactions = fmt.Sprintf("%s/v1/transactions?account=%s", url, a.ID)
}

type AccountListResponse struct {
	Data []Account `json:"data"` // List of accounts
}

type AccountResponse struct {
	Data Account `json:"data"` // Data for the account
}

func (co Controller) getAccount(c *gin.Context, id uuid.UUID) (Account, bool) {
	accountModel, ok := getResourceByIDAndHandleErrors[models.Account](c, co, id)

	account := Account{
		Account: accountModel,
	}

	if !ok {
		return Account{}, false
	}

	// Recent Envelopes
	envelopeIDs, err := accountModel.RecentEnvelopes(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return Account{}, false
	}

	envelopes := make([]models.Envelope, 0)
	for _, id := range envelopeIDs {
		// If the ID is nil, append the zero Envelope
		if id == nil {
			envelopes = append(envelopes, models.Envelope{})
			continue
		}

		envelope, ok := getResourceByIDAndHandleErrors[models.Envelope](c, co, *id)
		if !ok {
			return Account{}, false
		}
		envelopes = append(envelopes, envelope)
	}

	account.RecentEnvelopes = envelopes

	// Balance
	balance, _, err := accountModel.GetBalanceMonth(co.DB, types.Month{})
	if err != nil {
		httperrors.Handler(c, err)
		return Account{}, false
	}
	account.Balance = balance

	// Reconciled Balance
	reconciledBalance, err := accountModel.SumReconciled(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return Account{}, false
	}
	account.ReconciledBalance = reconciledBalance

	// Links
	account.links(c)

	return account, true
}

// RegisterAccountRoutes registers the routes for accounts with
// the RouterGroup that is passed.
func (co Controller) RegisterAccountRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsAccountList)
		r.GET("", co.GetAccounts)
		r.POST("", co.CreateAccount)
	}

	// Account with ID
	{
		r.OPTIONS("/:accountId", co.OptionsAccountDetail)
		r.GET("/:accountId", co.GetAccount)
		r.PATCH("/:accountId", co.UpdateAccount)
		r.DELETE("/:accountId", co.DeleteAccount)
	}
}

// OptionsAccountList returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Accounts
//	@Success		204
//	@Router			/v1/accounts [options]
func (co Controller) OptionsAccountList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsAccountDetail returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Accounts
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			accountId	path		string	true	"ID formatted as string"
//	@Router			/v1/accounts/{accountId} [options]
func (co Controller) OptionsAccountDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := co.getAccount(c, id)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// CreateAccount creates a new account
//
//	@Summary		Create account
//	@Description	Creates a new account
//	@Tags			Accounts
//	@Produce		json
//	@Success		201		{object}	AccountResponse
//	@Failure		400		{object}	httperrors.HTTPError
//	@Failure		404		{object}	httperrors.HTTPError
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			account	body		models.AccountCreate	true	"Account"
//	@Router			/v1/accounts [post]
func (co Controller) CreateAccount(c *gin.Context) {
	var accountCreate models.AccountCreate

	if err := httputil.BindData(c, &accountCreate); err != nil {
		return
	}

	account := models.Account{
		AccountCreate: accountCreate,
	}

	// Check if the budget that the account shoud belong to exists
	_, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, account.BudgetID)
	if !ok {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Create(&account)) {
		return
	}

	accountObject, ok := co.getAccount(c, account.ID)
	if !ok {
		return
	}
	c.JSON(http.StatusCreated, AccountResponse{Data: accountObject})
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
//	@Router			/v1/accounts [get]
//	@Param			name		query	string	false	"Filter by name"
//	@Param			note		query	string	false	"Filter by note"
//	@Param			budget		query	string	false	"Filter by budget ID"
//	@Param			onBudget	query	bool	false	"Is the account on-budget?"
//	@Param			external	query	bool	false	"Is the account external?"
//	@Param			hidden		query	bool	false	"Is the account hidden?"
//	@Param			search		query	string	false	"Search for this text in name and note"
//	@Deprecated		true
func (co Controller) GetAccounts(c *gin.Context) {
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
	accountObjects := make([]Account, 0)

	for _, account := range accounts {
		o, ok := co.getAccount(c, account.ID)
		if !ok {
			return
		}

		accountObjects = append(accountObjects, o)
	}

	c.JSON(http.StatusOK, AccountListResponse{Data: accountObjects})
}

// GetAccount returns data for a specific account
//
//	@Summary		Get account
//	@Description	Returns a specific account
//	@Tags			Accounts
//	@Produce		json
//	@Success		200			{object}	AccountResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			accountId	path		string	true	"ID formatted as string"
//	@Router			/v1/accounts/{accountId} [get]
func (co Controller) GetAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	accountObject, ok := co.getAccount(c, id)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, AccountResponse{Data: accountObject})
}

// UpdateAccount updates data for a specific account
//
//	@Summary		Update account
//	@Description	Updates an account. Only values to be updated need to be specified.
//	@Tags			Accounts
//	@Produce		json
//	@Success		200			{object}	AccountResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			accountId	path		string					true	"ID formatted as string"
//	@Param			account		body		models.AccountCreate	true	"Account"
//	@Router			/v1/accounts/{accountId} [patch]
func (co Controller) UpdateAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	account, ok := getResourceByIDAndHandleErrors[models.Account](c, co, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.AccountCreate{})
	if err != nil {
		return
	}

	var data models.Account
	if err := httputil.BindData(c, &data.AccountCreate); err != nil {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Model(&account).Select("", updateFields...).Updates(data)) {
		return
	}

	accountObject, ok := co.getAccount(c, account.ID)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, AccountResponse{Data: accountObject})
}

// DeleteAccount deletes an account
//
//	@Summary		Delete account
//	@Description	Deletes an account
//	@Tags			Accounts
//	@Produce		json
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			accountId	path		string	true	"ID formatted as string"
//	@Router			/v1/accounts/{accountId} [delete]
func (co Controller) DeleteAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	account, ok := getResourceByIDAndHandleErrors[models.Account](c, co, id)

	if !ok {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Delete(&account)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
