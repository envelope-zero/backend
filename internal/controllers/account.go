package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type AccountListResponse struct {
	Data []models.Account `json:"data"`
}

type AccountResponse struct {
	Data  models.Account `json:"data"`
	Links AccountLinks   `json:"links"`
}

type AccountLinks struct {
	Transactions string `json:"transactions" example:"https://example.com/api/v1/budgets/3/accounts/17/transactions"`
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

	// Transactions
	{
		r.OPTIONS("/:accountId/transactions", OptionsAccountTransactions)
		r.GET("/:accountId/transactions", GetAccountTransactions)
	}
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Accounts
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Param        budgetId   path      uint64                true  "ID of the budget"
// @Param        accountId  path      uint64                true  "ID of the account"
// @Router       /v1/budgets/{budgetId}/accounts/{accountId}/transactions [options]
func OptionsAccountTransactions(c *gin.Context) {
	httputil.OptionsGet(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Accounts
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Param        budgetId  path  uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId}/accounts [options]
func OptionsAccountList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Accounts
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Param        budgetId   path      uint64  true  "ID of the budget"
// @Param        accountId  path      uint64  true  "ID of the account"
// @Router       /v1/budgets/{budgetId}/accounts/{accountId} [options]
func OptionsAccountDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// @Summary      List all transactions for an account
// @Description  Returns a list of all transactions for the account requested
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}  TransactionListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500        {object}  httputil.HTTPError
// @Param        budgetId  path      uint64  true  "ID of the budget"
// @Param        accountId  path      uint64  true  "ID of the account"
// @Router       /v1/budgets/{budgetId}/accounts/{accountId}/transactions [get]
func GetAccountTransactions(c *gin.Context) {
	account, err := getAccountResource(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, TransactionListResponse{
		Data: account.Transactions(),
	})
}

// @Summary      Create account
// @Description  Create a new account for a specific budget
// @Tags         Accounts
// @Produce      json
// @Success      201  {object}  AccountResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500       {object}  httputil.HTTPError
// @Param        budgetId  path      uint64                true  "ID of the budget"
// @Param        account   body      models.AccountCreate  true  "Account"
// @Router       /v1/budgets/{budgetId}/accounts [post]
func CreateAccount(c *gin.Context) {
	var data models.Account

	if status, err := httputil.BindData(c, &data); err != nil {
		httputil.NewError(c, status, err)
		return
	}

	budget, err := getBudgetResource(c)
	if err != nil {
		return
	}

	data.BudgetID = budget.ID
	models.DB.Create(&data)

	c.JSON(http.StatusCreated, AccountResponse{Data: data})
}

// @Summary      List accounts
// @Description  Returns a list of all accounts for the budget
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}  AccountListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500       {object}  httputil.HTTPError
// @Param        budgetId   path      uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId}/accounts [get]
func GetAccounts(c *gin.Context) {
	var accounts []models.Account

	// Check if the budget exists at all
	budget, err := getBudgetResource(c)
	if err != nil {
		return
	}

	models.DB.Where(&models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
		},
	}).Find(&accounts)

	for i, account := range accounts {
		accounts[i] = account.WithCalculations()
	}

	c.JSON(http.StatusOK, AccountListResponse{Data: accounts})
}

// @Summary      Get account
// @Description  Returns a specific account
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}  AccountResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500        {object}  httputil.HTTPError
// @Param        budgetId   path      uint64  true  "ID of the budget"
// @Param        accountId  path      uint64  true  "ID of the account"
// @Router       /v1/budgets/{budgetId}/accounts/{accountId} [get]
func GetAccount(c *gin.Context) {
	_, err := getAccountResource(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, newAccountResponse(c))
}

// @Summary      Update account
// @Description  Updates an account. Only values to be updated need to be specified.
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}  AccountResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500        {object}  httputil.HTTPError
// @Param        budgetId   path  uint64  true  "ID of the budget"
// @Param        accountId  path  uint64  true  "ID of the account"
// @Param        account    body      models.AccountCreate  true  "Account"
// @Router       /v1/budgets/{budgetId}/accounts/{accountId} [patch]
func UpdateAccount(c *gin.Context) {
	account, err := getAccountResource(c)
	if err != nil {
		return
	}

	var data models.Account
	if status, err := httputil.BindData(c, &data); err != nil {
		httputil.NewError(c, status, err)
		return
	}

	models.DB.Model(&account).Updates(data)
	c.JSON(http.StatusOK, newAccountResponse(c))
}

// @Summary      Delete account
// @Description  Deletes the specified account.
// @Tags         Accounts
// @Produce      json
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500        {object}  httputil.HTTPError
// @Param        budgetId   path  uint64  true  "ID of the budget"
// @Param        accountId  path  uint64  true  "ID of the account"
// @Router       /v1/budgets/{budgetId}/accounts/{accountId} [delete]
func DeleteAccount(c *gin.Context) {
	account, err := getAccountResource(c)
	if err != nil {
		return
	}

	models.DB.Delete(&account)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getAccountResource verifies that the request URI is valid for the account and returns it.
func getAccountResource(c *gin.Context) (models.Account, error) {
	var account models.Account

	budget, err := getBudgetResource(c)
	if err != nil {
		return models.Account{}, err
	}

	accountID, err := httputil.ParseID(c, "accountId")
	if err != nil {
		return models.Account{}, err
	}

	err = models.DB.First(&account, &models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
		},
		Model: models.Model{
			ID: accountID,
		},
	}).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Account{}, err
	}

	return account, nil
}

// newAccountResponse creates a response object for an account.
func newAccountResponse(c *gin.Context) AccountResponse {
	// When this function is called, all parent resources have already been validated
	budget, _ := getBudgetResource(c)
	account, _ := getAccountResource(c)

	url := httputil.RequestPathV1(c) + fmt.Sprintf("/budgets/%d/accounts/%d", budget.ID, account.ID)

	return AccountResponse{
		Data: account.WithCalculations(),
		Links: AccountLinks{
			Transactions: url + "/transactions",
		},
	}
}
