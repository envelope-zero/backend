package v3

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

// RegisterAccountRoutes registers the routes for accounts with
// the RouterGroup that is passed.
func RegisterAccountRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsAccountList)
		r.GET("", GetAccounts)
		r.POST("", CreateAccounts)
	}

	// Account with ID
	{
		r.OPTIONS("/:id", OptionsAccountDetail)
		r.GET("/:id", GetAccount)
		r.PATCH("/:id", UpdateAccount)
		r.DELETE("/:id", DeleteAccount)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Accounts
// @Success		204
// @Router			/v3/accounts [options].
func OptionsAccountList(c *gin.Context) {
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
func OptionsAccountDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	var account models.Account
	err = query(c, models.DB.First(&account, id))
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
// @Success		201			{object}	AccountCreateResponse
// @Failure		400			{object}	AccountCreateResponse
// @Failure		404			{object}	AccountCreateResponse
// @Failure		500			{object}	AccountCreateResponse
// @Param			accounts	body		[]AccountEditable	true	"Accounts"
// @Router			/v3/accounts [post]
func CreateAccounts(c *gin.Context) {
	var editables []AccountEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &editables)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, TransactionCreateResponse{
			Error: &e,
		})
		return
	}
	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := AccountCreateResponse{}

	for _, editable := range editables {
		account := editable.model()

		// Verify that budget exists. If not, append the error
		// and move to the next account
		_, err := getResourceByID[models.Budget](c, editable.BudgetID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		dbErr := models.DB.Create(&account).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Account](account, c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		data, err := newAccount(c, models.DB, account)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}
		r.Data = append(r.Data, AccountResponse{Data: &data})
	}

	c.JSON(status, r)
}

// @Summary		List accounts
// @Description	Returns a list of accounts
// @Tags			Accounts
// @Produce		json
// @Success		200	{object}	AccountListResponse
// @Failure		400	{object}	AccountListResponse
// @Failure		500	{object}	AccountListResponse
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
func GetAccounts(c *gin.Context) {
	var filter AccountQueryFilter
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
		c.JSON(err.Status, AccountListResponse{
			Error: &s,
		})
		return
	}

	q := models.DB.
		Order("name ASC").
		Where(&model, queryFields...)

	q = stringFilters(models.DB, q, setFields, filter.Name, filter.Note, filter.Search)

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
		c.JSON(err.Status, AccountListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, AccountListResponse{
			Error: &e,
		})
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	data := make([]Account, 0)
	for _, account := range accounts {
		apiResource, err := newAccount(c, models.DB, account)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, AccountListResponse{
				Error: &s,
			})
		}

		data = append(data, apiResource)
	}

	c.JSON(http.StatusOK, AccountListResponse{
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
// @Success		200	{object}	AccountResponse
// @Failure		400	{object}	AccountResponse
// @Failure		404	{object}	AccountResponse
// @Failure		500	{object}	AccountResponse
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/accounts/{id} [get]
func GetAccount(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	var account models.Account
	err = query(c, models.DB.First(&account, id))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	data, err := newAccount(c, models.DB, account)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, AccountResponse{Data: &data})
}

// @Summary		Update account
// @Description	Updates an account. Only values to be updated need to be specified.
// @Tags			Accounts
// @Produce		json
// @Success		200		{object}	AccountResponse
// @Failure		400		{object}	AccountResponse
// @Failure		404		{object}	AccountResponse
// @Failure		500		{object}	AccountResponse
// @Param			id		path		string			true	"ID formatted as string"
// @Param			account	body		AccountEditable	true	"Account"
// @Router			/v3/accounts/{id} [patch]
func UpdateAccount(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	account, err := getResourceByID[models.Account](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, AccountEditable{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	var data AccountEditable
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	err = query(c, models.DB.Model(&account).Select("", updateFields...).Updates(data.model()))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	apiResource, err := newAccount(c, models.DB, account)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, AccountResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, AccountResponse{Data: &apiResource})
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
func DeleteAccount(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	account, err := getResourceByID[models.Account](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, models.DB.Delete(&account))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
