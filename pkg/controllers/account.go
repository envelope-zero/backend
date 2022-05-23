package controllers

import (
	"fmt"
	"net/http"

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
	Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions?=af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`
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
// @Param        accountId  path      uint64  true  "ID of the account"
// @Router       /v1/accounts/{accountId} [options]
func OptionsAccountDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// @Summary      Create account
// @Description  Create a new account
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

	models.DB.Create(&account)

	accountObject, _ := getAccountObject(c, account.ID)
	c.JSON(http.StatusCreated, AccountResponse{Data: accountObject})
}

// @Summary      List accounts
// @Description  Returns a list of all accounts
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}  AccountListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/accounts [get]
func GetAccounts(c *gin.Context) {
	var accounts []models.Account

	models.DB.Find(&accounts)

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
// @Param        accountId  path      uint64                true  "ID of the account"
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
// @Param        accountId  path      uint64  true  "ID of the account"
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

	models.DB.Model(&account).Updates(data)
	accountObject, _ := getAccountObject(c, account.ID)
	c.JSON(http.StatusOK, AccountResponse{Data: accountObject})
}

// @Summary      Delete account
// @Description  Deletes the specified account.
// @Tags         Accounts
// @Produce      json
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500        {object}  httputil.HTTPError
// @Param        accountId  path  uint64  true  "ID of the account"
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

	models.DB.Delete(&account)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getAccountResource is the internal helper to verify permissions and return an account.
func getAccountResource(c *gin.Context, id uuid.UUID) (models.Account, error) {
	var account models.Account

	err := models.DB.Where(&models.Account{
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
