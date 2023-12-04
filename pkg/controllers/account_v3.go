package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

// AccountCreateV3 represents all user configurable parameters
type AccountCreateV3 struct {
	Name               string          `json:"name" example:"Cash" default:"" gorm:"uniqueIndex:account_name_budget_id"`                          // Name of the account
	Note               string          `json:"note" example:"Money in my wallet" default:""`                                                      // A longer description for the account
	BudgetID           uuid.UUID       `json:"budgetId" example:"550dc009-cea6-4c12-b2a5-03446eb7b7cf" gorm:"uniqueIndex:account_name_budget_id"` // ID of the budget this account belongs to
	OnBudget           bool            `json:"onBudget" example:"true" default:"false"`                                                           // Does the account factor into the available budget? Always false when external: true
	External           bool            `json:"external" example:"false" default:"false"`                                                          // Does the account belong to the budget owner or not?
	InitialBalance     decimal.Decimal `json:"initialBalance" example:"173.12" default:"0"`                                                       // Balance of the account before any transactions were recorded
	InitialBalanceDate *time.Time      `json:"initialBalanceDate" example:"2017-05-12T00:00:00Z"`                                                 // Date of the initial balance
	Archived           bool            `json:"archived" example:"true" default:"false"`                                                           // Is the account archived?
	ImportHash         string          `json:"importHash" example:"867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70" default:""`  // The SHA256 hash of a unique combination of values to use in duplicate detection
}

// ToCreate transforms the API representation into the model representation
func (a AccountCreateV3) ToCreate() models.AccountCreate {
	return models.AccountCreate{
		Name:               a.Name,
		Note:               a.Note,
		BudgetID:           a.BudgetID,
		OnBudget:           a.OnBudget,
		External:           a.External,
		InitialBalance:     a.InitialBalance,
		InitialBalanceDate: a.InitialBalanceDate,
		Hidden:             a.Archived,
		ImportHash:         a.ImportHash,
	}
}

type AccountListResponseV3 struct {
	Data       []AccountV3 `json:"data"`                                                          // List of accounts
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type AccountCreateResponseV3 struct {
	Error *string             `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []AccountResponseV3 `json:"data"`                                                          // List of created Accounts
}

type AccountResponseV3 struct {
	Data  *AccountV3 `json:"data"`                                                          // Data for the account
	Error *string    `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this transaction
}

// AccountV3 is the API v3 representation of an Account in EZ.
type AccountV3 struct {
	models.Account
	Balance           decimal.Decimal `json:"balance" example:"2735.17"`           // Balance of the account, including all transactions referencing it
	ReconciledBalance decimal.Decimal `json:"reconciledBalance" example:"2539.57"` // Balance of the account, including all reconciled transactions referencing it
	RecentEnvelopes   []*uuid.UUID    `json:"recentEnvelopes"`                     // Envelopes recently used with this account
	Hidden            bool            `json:"hidden,omitempty"`                    // Remove the hidden field

	Links struct {
		Self         string `json:"self" example:"https://example.com/api/v3/accounts/af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`                     // The account itself
		Transactions string `json:"transactions" example:"https://example.com/api/v3/transactions?account=af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"` // Transactions referencing the account
	} `json:"links"`
}

// links generates the HATEOAS links for the Account.
func (a *AccountV3) links(c *gin.Context) {
	url := c.GetString(string(database.ContextURL))
	a.Links.Self = fmt.Sprintf("%s/v3/accounts/%s", url, a.ID)
	a.Links.Transactions = fmt.Sprintf("%s/v3/transactions?account=%s", url, a.ID)
}

type AccountQueryFilterV3 struct {
	Name     string `form:"name" filterField:"false"`     // Fuzzy filter for the account name
	Note     string `form:"note" filterField:"false"`     // Fuzzy filter for the note
	BudgetID string `form:"budget"`                       // By budget ID
	OnBudget bool   `form:"onBudget"`                     // Is the account on-budget?
	External bool   `form:"external"`                     // Is the account external?
	Archived bool   `form:"archived" filterField:"false"` // Is the account hidden?
	Search   string `form:"search" filterField:"false"`   // By string in name or note
	Offset   uint   `form:"offset" filterField:"false"`   // The offset of the first Account returned. Defaults to 0.
	Limit    int    `form:"limit" filterField:"false"`    // Maximum number of Accounts to return. Defaults to 50.
}

func (f AccountQueryFilterV3) ToCreate() (models.AccountCreate, httperrors.Error) {
	budgetID, err := httputil.UUIDFromString(f.BudgetID)
	if !err.Nil() {
		return models.AccountCreate{}, err
	}

	return models.AccountCreate{
		BudgetID: budgetID,
		OnBudget: f.OnBudget,
		External: f.External,
		Hidden:   f.Archived,
	}, httperrors.Error{}
}

func (co Controller) getAccountV3(c *gin.Context, id uuid.UUID) (AccountV3, httperrors.Error) {
	m, e := getResourceByID[models.Account](c, co, id)
	if !e.Nil() {
		return AccountV3{}, e
	}

	account := AccountV3{
		Account: m,
	}

	// Recent Envelopes
	ids, err := m.RecentEnvelopes(co.DB)
	if err != nil {
		e := httperrors.Parse(c, err)
		return AccountV3{}, e
	}

	account.RecentEnvelopes = ids

	// Balance
	balance, _, err := m.GetBalanceMonth(co.DB, types.Month{})
	if err != nil {
		e := httperrors.Parse(c, err)
		return AccountV3{}, e
	}
	account.Balance = balance

	// Reconciled Balance
	reconciledBalance, err := m.SumReconciled(co.DB)
	if err != nil {
		e := httperrors.Parse(c, err)
		return AccountV3{}, e
	}
	account.ReconciledBalance = reconciledBalance

	// Links
	account.links(c)

	return account, httperrors.Error{}
}

// RegisterAccountRoutesV3 registers the routes for accounts with
// the RouterGroup that is passed.
func (co Controller) RegisterAccountRoutesV3(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsAccountListV3)
		r.GET("", co.GetAccountsV3)
		r.POST("", co.CreateAccountsV3)
	}

	// Account with ID
	{
		r.OPTIONS("/:id", co.OptionsAccountDetailV3)
		r.GET("/:id", co.GetAccountV3)
		r.PATCH("/:id", co.UpdateAccountV3)
		r.DELETE("/:id", co.DeleteAccountV3)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Accounts
// @Success		204
// @Router			/v3/accounts [options].
func (co Controller) OptionsAccountListV3(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Accounts
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/accounts/{id} [options].
func (co Controller) OptionsAccountDetailV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	_, err = co.getAccountV3(c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// @Summary		Creates accounts
// @Description	Creates new accounts
// @Tags			Accounts
// @Produce		json
// @Success		201			{object}	AccountCreateResponseV3
// @Failure		400			{object}	AccountCreateResponseV3
// @Failure		404			{object}	AccountCreateResponseV3
// @Failure		500			{object}	AccountCreateResponseV3
// @Param			accounts	body		[]AccountCreateV3	true	"Accounts"
// @Router			/v3/accounts [post].
func (co Controller) CreateAccountsV3(c *gin.Context) {
	var accounts []AccountCreateV3

	// Bind data and return error if not possible
	err := httputil.BindData(c, &accounts)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionCreateResponseV3{
			Error: &e,
		})
		return
	}
	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := AccountCreateResponseV3{}

	for _, create := range accounts {
		a := models.Account{
			AccountCreate: create.ToCreate(),
		}

		dbErr := co.DB.Create(&a).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Account](a, c, dbErr)
			s := err.Error()
			c.JSON(err.Status, AccountCreateResponseV3{
				Error: &s,
			})
			return
		}

		// Append the error
		if !err.Nil() {
			e := err.Error()
			r.Data = append(r.Data, AccountResponseV3{Error: &e})

			// The final status code is the highest HTTP status code number since this also
			// represents the priority we
			if err.Status > status {
				status = err.Status
			}
			continue
		}

		aObject, err := co.getAccountV3(c, a.ID)
		if !err.Nil() {
			e := err.Error()
			c.JSON(err.Status, AccountCreateResponseV3{
				Error: &e,
			})
			return
		}
		r.Data = append(r.Data, AccountResponseV3{Data: &aObject})
	}

	c.JSON(status, r)
}

// @Summary		List accounts
// @Description	Returns a list of accounts
// @Tags			Accounts
// @Produce		json
// @Success		200	{object}	AccountListResponseV3
// @Failure		400	{object}	AccountListResponseV3
// @Failure		500	{object}	AccountListResponseV3
// @Router			/v3/accounts [get]
// @Param			name		query	string	false	"Filter by name"
// @Param			note		query	string	false	"Filter by note"
// @Param			budget		query	string	false	"Filter by budget ID"
// @Param			onBudget	query	bool	false	"Is the account on-budget?"
// @Param			external	query	bool	false	"Is the account external?"
// @Param			archived	query	bool	false	"Is the account archived?"
// @Param			search		query	string	false	"Search for this text in name and note"
// @Param			offset		query	uint	false	"The offset of the first Account returned. Defaults to 0."
// @Param			limit		query	int		false	"Maximum number of Accounts to return. Defaults to 50."
func (co Controller) GetAccountsV3(c *gin.Context) {
	var filter AccountQueryFilterV3
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the set parameters in the query string
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// If the archived parameter is set, add "Hidden" to the query fields
	// This is done since in v3, we're using the name "Archived", but the
	// field is not yet updated in the database, which will happen later
	if slices.Contains(setFields, "Archived") {
		queryFields = append(queryFields, "Hidden")
	}

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate()
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountListResponseV3{
			Error: &s,
		})
		return
	}

	q := co.DB.
		Order("name ASC").
		Where(&models.Account{
			AccountCreate: create,
		}, queryFields...)

	q = stringFilters(co.DB, q, setFields, filter.Name, filter.Note, filter.Search)

	// Set the offset. Does not need checking since the default is 0
	q = q.Offset(int(filter.Offset))

	// Default to 50 Accounts and set the limit
	limit := 50
	if slices.Contains(setFields, "Limit") {
		limit = filter.Limit
	}
	q = q.Limit(limit)

	var accounts []models.Account
	err = query(c, q.Find(&accounts))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountListResponseV3{
			Error: &s,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, AccountListResponseV3{
			Error: &e,
		})
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	accountObjects := make([]AccountV3, 0)
	for _, account := range accounts {
		o, err := co.getAccountV3(c, account.ID)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, AccountListResponseV3{
				Error: &s,
			})
		}

		accountObjects = append(accountObjects, o)
	}

	c.JSON(http.StatusOK, AccountListResponseV3{
		Data: accountObjects,
		Pagination: &Pagination{
			Count:  len(accountObjects),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// @Summary		Get account
// @Description	Returns a specific account
// @Tags			Accounts
// @Produce		json
// @Success		200	{object}	AccountResponseV3
// @Failure		400	{object}	AccountResponseV3
// @Failure		404	{object}	AccountResponseV3
// @Failure		500	{object}	AccountResponseV3
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/accounts/{id} [get]
func (co Controller) GetAccountV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	accountObject, err := co.getAccountV3(c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, AccountResponseV3{Data: &accountObject})
}

// @Summary		Update account
// @Description	Updates an account. Only values to be updated need to be specified.
// @Tags			Accounts
// @Produce		json
// @Success		200		{object}	AccountResponseV3
// @Failure		400		{object}	AccountResponseV3
// @Failure		404		{object}	AccountResponseV3
// @Failure		500		{object}	AccountResponseV3
// @Param			id		path		string			true	"ID formatted as string"
// @Param			account	body		AccountCreateV3	true	"Account"
// @Router			/v3/accounts/{id} [patch]
func (co Controller) UpdateAccountV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	account, err := getResourceByID[models.Account](c, co, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, AccountCreateV3{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	var data AccountCreateV3
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	// Transform the API representation to the model representation
	a := models.Account{
		AccountCreate: data.ToCreate(),
	}

	err = query(c, co.DB.Model(&account).Select("", updateFields...).Updates(a))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	accountObject, err := co.getAccountV3(c, account.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, AccountResponseV3{Data: &accountObject})
}

// @Summary		Delete account
// @Description	Deletes an account
// @Tags			Accounts
// @Produce		json
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/accounts/{id} [delete]
func (co Controller) DeleteAccountV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	account, err := getResourceByID[models.Account](c, co, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, co.DB.Delete(&account))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
