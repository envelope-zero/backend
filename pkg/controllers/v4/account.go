package v4

import (
	"net/http"

	"github.com/envelope-zero/backend/v5/internal/uuid"
	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
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
		r.GET("/:id/recent-envelopes", GetAccountRecentEnvelopes)
		r.POST("/computed", GetAccountData) // This is a POST endpoints because some clients don't allow GET requests to have bodies
		r.PATCH("/:id", UpdateAccount)
		r.DELETE("/:id", DeleteAccount)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Accounts
// @Success		204
// @Router			/v4/accounts [options]
func OptionsAccountList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Accounts
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/accounts/{id} [options]
func OptionsAccountDetail(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.First(&models.Account{}, uri.ID).Error
	if err != nil {
		c.JSON(status(err), httpError{
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
// @Router			/v4/accounts [post]
func CreateAccounts(c *gin.Context) {
	var editables []AccountEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &editables)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), AccountCreateResponse{
			Error: &e,
		})
		return
	}
	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := AccountCreateResponse{}

	for _, editable := range editables {
		account := editable.model()
		err = models.DB.Create(&account).Error
		if err != nil {
			status = r.appendError(err, status)
			continue
		}

		data := newAccount(c, account)
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
// @Router			/v4/accounts [get]
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
		s := err.Error()
		c.JSON(http.StatusBadRequest, AccountListResponse{
			Error: &s,
		})
		return
	}

	// Get the set parameters in the query string
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	model, err := filter.model()
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountListResponse{
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
	err = q.Find(&accounts).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = q.Limit(-1).Offset(-1).Count(&count).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), AccountListResponse{
			Error: &e,
		})
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	data := make([]Account, 0)
	for _, account := range accounts {
		data = append(data, newAccount(c, account))
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
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/accounts/{id} [get]
func GetAccount(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountResponse{
			Error: &s,
		})
		return
	}

	var account models.Account
	err = models.DB.First(&account, uri.ID).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountResponse{
			Error: &s,
		})
		return
	}

	data := newAccount(c, account)
	c.JSON(http.StatusOK, AccountResponse{Data: &data})
}

//	@Summary		Get recent envelopes
//	@Description	Returns a list of objects representing recent envelopes
//	@Tags			Accounts
//	@Produce		json
//	@Success		200	{object}	RecentEnvelopesResponse
//	@Failure		400	{object}	RecentEnvelopesResponse
//	@Failure		404	{object}	RecentEnvelopesResponse
//	@Failure		500	{object}	RecentEnvelopesResponse
//	@Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
//	@Router			/v4/accounts/{id}/recent-envelopes [get]
//
// GetAccountRecentEnvelopes returns recent envelopes for an account.
//
// Income is returned as a RecentEnvelope with the nil ID.
// Clients must be able to handle this.
func GetAccountRecentEnvelopes(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), RecentEnvelopesResponse{
			Error: &s,
		})
		return
	}

	var account models.Account
	err = models.DB.First(&account, uri.ID).Error
	if err != nil {
		s := err.Error()

		c.JSON(status(err), RecentEnvelopesResponse{
			Error: &s,
		})
		return
	}

	var recentEnvelopes []RecentEnvelope

	// Get the Envelope IDs for the 50 latest transactions
	latest := models.DB.
		Model(&models.Transaction{}).
		Joins("LEFT JOIN envelopes ON envelopes.id = transactions.envelope_id AND envelopes.deleted_at IS NULL").
		Select("envelopes.id as e_id, envelopes.name as name, datetime(envelopes.created_at) as created, envelopes.archived as archived").
		Where(&models.Transaction{
			DestinationAccountID: account.ID,
		}).
		Order("datetime(transactions.date) DESC").
		Limit(50)

	// Group by frequency
	err = models.DB.
		Table("(?)", latest).
		// Set the nil UUID as ID if the envelope ID is NULL, since count() only counts non-null values
		Select("IIF(e_id IS NOT NULL, e_id, NULL) as id, name, archived").
		Group("id").
		Order("count(IIF(e_id IS NOT NULL, e_id, '0')) DESC"). // Order with a different IIF since NULL is ignored for count
		Order("created ASC").
		Limit(5).
		Find(&recentEnvelopes).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), RecentEnvelopesResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, RecentEnvelopesResponse{Data: recentEnvelopes})
}

// @Summary		Get Account data
// @Description	Returns calculated data for the account, e.g. balances
// @Tags			Accounts
// @Produce		json
// @Success		200		{object}	AccountComputedDataResponse
// @Failure		400		{object}	AccountComputedDataResponse
// @Failure		404		{object}	AccountComputedDataResponse
// @Failure		500		{object}	AccountComputedDataResponse
// @Param			request	body		AccountComputedRequest	true	"Time and IDs of requested accounts"
// @Router			/v4/accounts/computed [post]
func GetAccountData(c *gin.Context) {
	var request AccountComputedRequest

	// Bind data and return error if not possible
	err := httputil.BindData(c, &request)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), AccountComputedDataResponse{
			Error: &e,
		})
		return
	}

	data := make([]AccountComputedData, 0)
	for _, idString := range request.IDs {
		var id uuid.UUID
		err := id.UnmarshalParam(idString)
		if err != nil {
			s := err.Error()
			c.JSON(status(err), AccountComputedDataResponse{
				Error: &s,
			})
			return
		}

		var account models.Account
		err = models.DB.First(&account, id).Error
		if err != nil {
			s := err.Error()
			c.JSON(status(err), AccountComputedDataResponse{
				Error: &s,
			})
			return
		}

		// Balance
		balance, err := account.Balance(models.DB, request.Time)
		if err != nil {
			s := err.Error()
			c.JSON(status(err), AccountComputedDataResponse{
				Error: &s,
			})
			return
		}

		// Reconciled Balance
		reconciledBalance, err := account.ReconciledBalance(models.DB, request.Time)
		if err != nil {
			s := err.Error()
			c.JSON(status(err), AccountComputedDataResponse{
				Error: &s,
			})
			return
		}

		data = append(data, AccountComputedData{
			ID:                id,
			Balance:           balance,
			ReconciledBalance: reconciledBalance,
		})
	}

	c.JSON(http.StatusOK, AccountComputedDataResponse{Data: data})
}

// @Summary		Update account
// @Description	Updates an account. Only values to be updated need to be specified.
// @Tags			Accounts
// @Produce		json
// @Success		200		{object}	AccountResponse
// @Failure		400		{object}	AccountResponse
// @Failure		404		{object}	AccountResponse
// @Failure		500		{object}	AccountResponse
// @Param			id		path		URIID			true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			account	body		AccountEditable	true	"Account"
// @Router			/v4/accounts/{id} [patch]
func UpdateAccount(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountResponse{
			Error: &s,
		})
		return
	}

	var account models.Account
	err = models.DB.First(&account, uri.ID).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, AccountEditable{})
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountResponse{
			Error: &s,
		})
		return
	}

	var data AccountEditable
	err = httputil.BindData(c, &data)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountResponse{
			Error: &s,
		})
		return
	}

	err = models.DB.Model(&account).Select("", updateFields...).Updates(data.model()).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), AccountResponse{
			Error: &s,
		})
		return
	}

	apiResource := newAccount(c, account)
	c.JSON(http.StatusOK, AccountResponse{Data: &apiResource})
}

// @Summary		Delete account
// @Description	Deletes an account
// @Tags			Accounts
// @Produce		json
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/accounts/{id} [delete]
func DeleteAccount(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var account models.Account
	err = models.DB.First(&account, uri.ID).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.Delete(&account).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
