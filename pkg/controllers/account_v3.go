package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

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

	var account models.Account
	err = query(c, co.DB.First(&account, id))
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
// @Param			accounts	body		[]AccountV3Editable	true	"Accounts"
// @Router			/v3/accounts [post]
func (co Controller) CreateAccountsV3(c *gin.Context) {
	var editables []AccountV3Editable

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
	r := AccountCreateResponseV3{}

	for _, editable := range editables {
		account := editable.model()

		// Verify that budget exists. If not, append the error
		// and move to the next account
		_, err := getResourceByID[models.Budget](c, co, editable.BudgetID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		dbErr := co.DB.Create(&account).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Account](account, c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		data, err := newAccountV3(c, co.DB, account)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}
		r.Data = append(r.Data, AccountResponseV3{Data: &data})
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

	// Convert the QueryFilter to a Create struct
	model, err := filter.model()
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountListResponseV3{
			Error: &s,
		})
		return
	}

	q := co.DB.
		Order("name ASC").
		Where(&model, queryFields...)

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
	data := make([]AccountV3, 0)
	for _, account := range accounts {
		apiResource, err := newAccountV3(c, co.DB, account)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, AccountListResponseV3{
				Error: &s,
			})
		}

		data = append(data, apiResource)
	}

	c.JSON(http.StatusOK, AccountListResponseV3{
		Data: data,
		Pagination: &Pagination{
			Count:  len(data),
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

	var account models.Account
	err = query(c, co.DB.First(&account, id))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	data, err := newAccountV3(c, co.DB, account)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, AccountResponseV3{Data: &data})
}

// @Summary		Update account
// @Description	Updates an account. Only values to be updated need to be specified.
// @Tags			Accounts
// @Produce		json
// @Success		200		{object}	AccountResponseV3
// @Failure		400		{object}	AccountResponseV3
// @Failure		404		{object}	AccountResponseV3
// @Failure		500		{object}	AccountResponseV3
// @Param			id		path		string				true	"ID formatted as string"
// @Param			account	body		AccountV3Editable	true	"Account"
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

	updateFields, err := httputil.GetBodyFields(c, AccountV3Editable{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	var data AccountV3Editable
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	err = query(c, co.DB.Model(&account).Select("", updateFields...).Updates(data.model()))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	apiResource, err := newAccountV3(c, co.DB, account)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, AccountResponseV3{Data: &apiResource})
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
