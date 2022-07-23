package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
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
	Links AccountLinks `json:"links"`
}

type AccountLinks struct {
	Self         string `json:"self" example:"https://example.com/api/v1/accounts/af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`
	Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions?account=af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`
}

type AccountQueryFilter struct {
	Name     string `form:"name"`
	Note     string `form:"note"`
	BudgetID string `form:"budget"`
	OnBudget bool   `form:"onBudget"`
	External bool   `form:"external"`
}

func (a AccountQueryFilter) ToCreate(c *gin.Context) (models.AccountCreate, error) {
	budgetID, err := httputil.UUIDFromString(c, a.BudgetID)
	if err != nil {
		return models.AccountCreate{}, err
	}

	return models.AccountCreate{
		Name:     a.Name,
		Note:     a.Note,
		BudgetID: budgetID,
		OnBudget: a.OnBudget,
		External: a.External,
	}, nil
}

// RegisterAccountRoutes registers the routes for accounts with
// the RouterGroup that is passed.
func RegisterAccountRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsAccountList)
		r.GET("", GetAccounts)
		r.POST("", CreateAccount)
	}

	// Account with ID
	{
		r.OPTIONS("/:accountId", OptionsAccountDetail)
		r.GET("/:accountId", GetAccount)
		r.PATCH("/:accountId", UpdateAccount)
		r.DELETE("/:accountId", DeleteAccount)
	}
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Accounts
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Router       /v1/accounts [options]
func OptionsAccountList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Accounts
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Param        accountId  path  string  true  "ID formatted as string"
// @Router       /v1/accounts/{accountId} [options]
func OptionsAccountDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// @Summary      Create account
// @Description  Creates a new account
// @Tags         Accounts
// @Produce      json
// @Success      201  {object}  AccountResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500      {object}  httputil.HTTPError
// @Param        account  body      models.AccountCreate  true  "Account"
// @Router       /v1/accounts [post]
func CreateAccount(c *gin.Context) {
	var account models.Account

	if err := httputil.BindData(c, &account); err != nil {
		return
	}

	// Check if the budget that the account shoud belong to exists
	_, err := getBudgetResource(c, account.BudgetID)
	if err != nil {
		return
	}

	database.DB.Create(&account)

	accountObject, _ := getAccountObject(c, account.ID)
	c.JSON(http.StatusCreated, AccountResponse{Data: accountObject})
}

// @Summary      List accounts
// @Description  Returns a list of accounts
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}  AccountListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/accounts [get]
// @Param        name      query  string  false  "Filter by name"
// @Param        note      query  string  false  "Filter by note"
// @Param        budget    query  string  false  "Filter by budget ID"
// @Param        onBudget  query  bool    false  "Filter by on/off-budget"
// @Param        external  query  bool    false  "Filter internal/external"
func GetAccounts(c *gin.Context) {
	var filter AccountQueryFilter
	if err := c.Bind(&filter); err != nil {
		httputil.ErrorInvalidQueryString(c)
		return
	}

	// Get the set parameters in the query string
	queryFields := httputil.GetFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate(c)
	if err != nil {
		return
	}

	var accounts []models.Account
	database.DB.Where(&models.Account{
		AccountCreate: create,
	}, queryFields...).Find(&accounts)

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	accountObjects := make([]Account, 0)

	for _, account := range accounts {
		o, _ := getAccountObject(c, account.ID)
		accountObjects = append(accountObjects, o)
	}

	c.JSON(http.StatusOK, AccountListResponse{Data: accountObjects})
}

// @Summary      Get account
// @Description  Returns a specific account
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}  AccountResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500        {object}  httputil.HTTPError
// @Param        accountId  path      string  true  "ID formatted as string"
// @Router       /v1/accounts/{accountId} [get]
func GetAccount(c *gin.Context) {
	p, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	accountObject, err := getAccountObject(c, p)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, AccountResponse{Data: accountObject})
}

// @Summary      Update account
// @Description  Updates an account. Only values to be updated need to be specified.
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}  AccountResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500        {object}  httputil.HTTPError
// @Param        accountId  path      string                true  "ID formatted as string"
// @Param        account    body      models.AccountCreate  true  "Account"
// @Router       /v1/accounts/{accountId} [patch]
func UpdateAccount(c *gin.Context) {
	p, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	account, err := getAccountResource(c, p)
	if err != nil {
		return
	}

	var data models.Account
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	database.DB.Model(&account).Updates(data)
	accountObject, _ := getAccountObject(c, account.ID)
	c.JSON(http.StatusOK, AccountResponse{Data: accountObject})
}

// @Summary      Delete account
// @Description  Deletes an account
// @Tags         Accounts
// @Produce      json
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500        {object}  httputil.HTTPError
// @Param        accountId  path      string  true  "ID formatted as string"
// @Router       /v1/accounts/{accountId} [delete]
func DeleteAccount(c *gin.Context) {
	p, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	account, err := getAccountResource(c, p)
	if err != nil {
		return
	}

	database.DB.Delete(&account)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getAccountResource is the internal helper to verify permissions and return an account.
func getAccountResource(c *gin.Context, id uuid.UUID) (models.Account, error) {
	if id == uuid.Nil {
		err := errors.New("No account ID specified")
		httputil.NewError(c, http.StatusBadRequest, err)
		return models.Account{}, err
	}

	var account models.Account

	err := database.DB.Where(&models.Account{
		Model: models.Model{
			ID: id,
		},
	}).First(&account).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Account{}, err
	}

	return account, nil
}

func getAccountObject(c *gin.Context, id uuid.UUID) (Account, error) {
	resource, err := getAccountResource(c, id)
	if err != nil {
		return Account{}, err
	}

	return Account{
		resource.WithCalculations(),
		getAccountLinks(c, resource.ID),
	}, nil
}

// getAccountLinks returns an AccountLinks struct.
//
// This function is only needed for getAccountObject as we cannot create an instance of Account
// with mixed named and unnamed parameters.
func getAccountLinks(c *gin.Context, id uuid.UUID) AccountLinks {
	url := httputil.RequestPathV1(c) + fmt.Sprintf("/accounts/%s", id)
	t := httputil.RequestPathV1(c) + fmt.Sprintf("/transactions?account=%s", id)

	return AccountLinks{
		Self:         url,
		Transactions: t,
	}
}
