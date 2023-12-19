package controllers

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/importer"
	ynabimport "github.com/envelope-zero/backend/v3/pkg/importer/parser/ynab-import"
	"github.com/envelope-zero/backend/v3/pkg/importer/parser/ynab4"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ImportPreviewList struct {
	Data []importer.TransactionPreview `json:"data"` // List of transaction previews
}

// RegisterImportRoutes registers the routes for imports.
func (co Controller) RegisterImportRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsImport)
		r.POST("", co.Import)

		r.OPTIONS("/ynab4", co.OptionsImportYnab4)
		r.POST("/ynab4", co.ImportYnab4)

		r.OPTIONS("/ynab-import-preview", co.OptionsImportYnabImportPreview)
		r.POST("/ynab-import-preview", co.ImportYnabImportPreview)
	}
}

// OptionsImport returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs. **Please use /v1/import/ynab4, which works exactly the same.**
//	@Tags			Import
//	@Success		204
//	@Router			/v1/import [options]
//	@Deprecated		true
func (co Controller) OptionsImport(c *gin.Context) {
	httputil.OptionsPost(c)
}

// OptionsImportYnab4 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Import
//	@Success		204
//	@Router			/v1/import/ynab4 [options]
//	@Deprecated		true
func (co Controller) OptionsImportYnab4(c *gin.Context) {
	httputil.OptionsPost(c)
}

// OptionsImportYnab4 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Import
//	@Success		204
//	@Router			/v1/import/ynab-import-preview [options]
//	@Deprecated		true
func (co Controller) OptionsImportYnabImportPreview(c *gin.Context) {
	httputil.OptionsPost(c)
}

// Import imports a YNAB 4 budget
//
//	@Summary		Import
//	@Description	Imports budgets from YNAB 4. **Please use /v1/import/ynab4, which works exactly the same.**
//	@Tags			Import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			file		formData	file	true	"File to import"
//	@Param			budgetName	query		string	false	"Name of the Budget to create"
//	@Router			/v1/import [post]
//	@Deprecated		true
func (co Controller) Import(c *gin.Context) {
	co.ImportYnab4(c)
}

// ImportYnabImportPreview parses a YNAB import format CSV and returns a preview of transactions
// to be imported into Envelope Zero.
//
//	@Summary		Transaction Import Preview
//	@Description	Returns a preview of transactions to be imported after parsing a YNAB Import format csv file
//	@Tags			Import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		200			{object}	ImportPreviewList
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			file		formData	file	true	"File to import"
//	@Param			accountId	query		string	false	"ID of the account to import transactions for"
//	@Router			/v1/import/ynab-import-preview [post]
//	@Deprecated		true
func (co Controller) ImportYnabImportPreview(c *gin.Context) {
	var query ImportPreviewQuery
	if err := c.BindQuery(&query); err != nil {
		httperrors.New(c, http.StatusBadRequest, httperrors.ErrAccountIDParameter.Error())
		return
	}

	f, e := getUploadedFile(c, ".csv")
	if !e.Nil() {
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	accountID, e := httputil.UUIDFromString(query.AccountID)
	if !e.Nil() {
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	// Verify that the account exists
	account, e := getResourceByID[models.Account](c, co, accountID)
	if !e.Nil() {
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	transactions, err := ynabimport.Parse(f, account)
	if err != nil {
		// ynabimport.Parse parsing returns a usable error already, no parsing necessary
		c.JSON(http.StatusBadRequest, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	// Get all match rules for the budget that the import target account is part of
	var matchRules []models.MatchRule
	err = co.DB.
		Joins("JOIN accounts ON accounts.budget_id = ?", account.BudgetID).
		Joins("JOIN match_rules rr ON rr.account_id = accounts.id").
		Order("rr.priority asc").
		Find(&matchRules).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	for i, transaction := range transactions {
		if len(matchRules) > 0 {
			match(&transaction, matchRules)
		}

		// Only find accounts when they are not yet both set
		if transaction.Transaction.SourceAccountID == uuid.Nil || transaction.Transaction.DestinationAccountID == uuid.Nil {
			err = findAccounts(co, &transaction, account.BudgetID)
			if err != nil {
				httperrors.Handler(c, err)
				return
			}
		}

		duplicateTransactions(co, &transaction, account.BudgetID)

		// Recommend an envelope
		if transaction.Transaction.DestinationAccountID != uuid.Nil {
			err = recommendEnvelope(co, &transaction, transaction.Transaction.DestinationAccountID)
			if err != nil {
				httperrors.Handler(c, err)
			}
		}

		transactions[i] = transaction
	}

	c.JSON(http.StatusOK, ImportPreviewList{Data: transactions})
}

// ImportYnab4 imports a YNAB 4 budget
//
//	@Summary		Import YNAB 4 budget
//	@Description	Imports budgets from YNAB 4
//	@Tags			Import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		201			{object}	BudgetResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			file		formData	file	true	"File to import"
//	@Param			budgetName	query		string	false	"Name of the Budget to create"
//	@Router			/v1/import/ynab4 [post]
//	@Deprecated		true
func (co Controller) ImportYnab4(c *gin.Context) {
	var query ImportQuery
	if err := c.BindQuery(&query); err != nil {
		httperrors.New(c, http.StatusBadRequest, "The budgetName parameter must be set")
		return
	}

	// Verify if the budget does already exist. If yes, return an error
	// as we only allow imports to new budgets
	var budget models.Budget
	err := co.DB.Where(&models.Budget{
		BudgetCreate: models.BudgetCreate{
			Name: query.BudgetName,
		},
	}).First(&budget).Error

	if err == nil {
		httperrors.New(c, http.StatusBadRequest, "This budget name is already in use. Imports from YNAB 4 create a new budget, therefore the name needs to be unique.")
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		httperrors.Handler(c, err)
		return
	}

	f, e := getUploadedFile(c, ".yfull")
	if !e.Nil() {
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	// Parse the Budget.yfull
	resources, err := ynab4.Parse(f)
	if err != nil {
		httperrors.New(c, http.StatusBadRequest, err.Error())
		return
	}

	// Set the budget name explicitly since YNAB 4 files
	// do not contain it
	resources.Budget.BudgetCreate.Name = query.BudgetName

	budget, err = importer.Create(co.DB, resources)
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	r, ok := co.getBudget(c, budget.ID)
	if !ok {
		return
	}

	c.JSON(http.StatusCreated, BudgetResponse{Data: r})
}