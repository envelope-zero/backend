package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/envelope-zero/backend/v2/pkg/httputil"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AccountListResponse struct {
	Data []Account `json:"data"`
}

type AccountResponse struct {
	Data Account `json:"data"`
}

type Account struct {
	models.Account
	RecentEnvelopes []models.Envelope `json:"recentEnvelopes"`
}

type AccountQueryFilter struct {
	Name     string `form:"name" filterField:"false"` // Fuzzy filter for the account name
	Note     string `form:"note" filterField:"false"` // Fuzzy filter for the note
	BudgetID string `form:"budget"`                   // By budget ID
	OnBudget bool   `form:"onBudget"`                 // Is the account on-budget?
	External bool   `form:"external"`                 // Is the account external?
	Hidden   bool   `form:"hidden"`                   // Is the account hidden?
	Search   string `form:"search" filterField:"false"`
}

func (f AccountQueryFilter) ToCreate(c *gin.Context) (models.AccountCreate, bool) {
	budgetID, ok := httputil.UUIDFromString(c, f.BudgetID)
	if !ok {
		return models.AccountCreate{}, false
	}

	return models.AccountCreate{
		BudgetID: budgetID,
		OnBudget: f.OnBudget,
		External: f.External,
		Hidden:   f.Hidden,
	}, true
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
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
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
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Param			accountId	path	string	true	"ID formatted as string"
//	@Router			/v1/accounts/{accountId} [options]
func (co Controller) OptionsAccountDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := co.getAccountObject(c, id)
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
//	@Success		201	{object}	AccountResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			account	body		models.AccountCreate	true	"Account"
//	@Router			/v1/accounts [post]
func (co Controller) CreateAccount(c *gin.Context) {
	var account models.Account

	if err := httputil.BindData(c, &account); err != nil {
		return
	}

	// Check if the budget that the account shoud belong to exists
	_, ok := getResourceByID[models.Budget](c, co, account.BudgetID)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Create(&account)) {
		return
	}

	accountObject, _ := co.getAccountObject(c, account.ID)
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
//	@Failure		404
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/accounts [get]
//	@Param			name		query	string	false	"Filter by name"
//	@Param			note		query	string	false	"Filter by note"
//	@Param			budget		query	string	false	"Filter by budget ID"
//	@Param			onBudget	query	bool	false	"Is the account on-budget?"
//	@Param			external	query	bool	false	"Is the account external?"
//	@Param			hidden		query	bool	false	"Is the account hidden?"
//	@Param			search		query	string	false	"Search for this text in name and note"
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
	if !queryWithRetry(c, query.Find(&accounts)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	accountObjects := make([]Account, 0)

	for _, account := range accounts {
		o, _ := co.getAccountObject(c, account.ID)
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
//	@Success		200	{object}	AccountResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			accountId	path		string	true	"ID formatted as string"
//	@Router			/v1/accounts/{accountId} [get]
func (co Controller) GetAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	accountObject, ok := co.getAccountObject(c, id)
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
//	@Success		200	{object}	AccountResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
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

	account, ok := co.getAccountResource(c, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.AccountCreate{})
	if err != nil {
		return
	}

	var data models.Account
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	if !queryWithRetry(c, co.DB.Model(&account).Select("", updateFields...).Updates(data)) {
		return
	}

	accountObject, _ := co.getAccountObject(c, account.ID)
	c.JSON(http.StatusOK, AccountResponse{Data: accountObject})
}

// DeleteAccount deletes an account
//
//	@Summary		Delete account
//	@Description	Deletes an account
//	@Tags			Accounts
//	@Produce		json
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			accountId	path		string	true	"ID formatted as string"
//	@Router			/v1/accounts/{accountId} [delete]
func (co Controller) DeleteAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	account, ok := co.getAccountResource(c, id)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&account)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// getAccountResource is the internal helper to verify permissions and return an account.
func (co Controller) getAccountResource(c *gin.Context, id uuid.UUID) (models.Account, bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "no account ID specified")
		return models.Account{}, false
	}

	var account models.Account

	if !queryWithRetry(c, co.DB.Where(&models.Account{
		DefaultModel: models.DefaultModel{
			ID: id,
		},
	}).First(&account), "No account found for the specified ID") {
		return models.Account{}, false
	}

	return account, true
}

func (co Controller) getAccountObject(c *gin.Context, id uuid.UUID) (Account, bool) {
	account, ok := co.getAccountResource(c, id)
	if !ok {
		return Account{}, false
	}

	recentEnvelopes, err := account.RecentEnvelopes(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return Account{}, false
	}

	account, err = account.WithCalculations(co.DB)
	if err != nil {
		httperrors.Handler(c, err)
		return Account{}, false
	}

	return Account{
		account,
		recentEnvelopes,
	}, true
}
