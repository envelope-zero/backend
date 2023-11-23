package controllers

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/database"
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

type ImportPreviewListV3 struct {
	Data  []importer.TransactionPreview `json:"data"`                                                          // List of transaction previews
	Error *string                       `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this Match Rule
}

// RegisterImportRoutes registers the routes for imports.
func (co Controller) RegisterImportRoutesV3(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsImportV3)
		r.GET("", co.GetImportV3)

		r.OPTIONS("/ynab4", co.OptionsImportYnab4V3)
		r.POST("/ynab4", co.ImportYnab4V3)

		r.OPTIONS("/ynab-import-preview", co.OptionsImportYnabImportPreviewV3)
		r.POST("/ynab-import-preview", co.ImportYnabImportPreviewV3)
	}
}

// OptionsImport returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs.
//	@Tags			Import
//	@Success		204
//	@Router			/v3/import [options]
func (co Controller) OptionsImportV3(c *gin.Context) {
	httputil.OptionsGet(c)
}

type ImportV3Response struct {
	Links ImportV3Links `json:"links"` // Links for the v3 API
}

type ImportV3Links struct {
	Ynab4             string `json:"transactions" example:"https://example.com/api/v3/import/ynab4"`             // URL of YNAB4 import endpoint
	YnabImportPreview string `json:"matchRules" example:"https://example.com/api/v3/import/ynab-import-preview"` // URL of YNAB Import preview endpoint
}

// GetImportV3 returns the link list for v3 Import API
//
//	@Summary		Import API overview
//	@Description	Returns general information about the v3 API
//	@Tags			Import
//	@Success		200	{object}	ImportV3Response
//	@Router			/v3/import [get]
func (Controller) GetImportV3(c *gin.Context) {
	c.JSON(http.StatusOK, ImportV3Response{
		Links: ImportV3Links{
			Ynab4:             c.GetString(string(database.ContextURL)) + "/v3/import/ynab4",
			YnabImportPreview: c.GetString(string(database.ContextURL)) + "/v3/import/ynab-import-preview",
		},
	})
}

// OptionsImportYnab4 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Import
//	@Success		204
//	@Router			/v3/import/ynab4 [options]
func (co Controller) OptionsImportYnab4V3(c *gin.Context) {
	httputil.OptionsPost(c)
}

// OptionsImportYnab4 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Import
//	@Success		204
//	@Router			/v3/import/ynab-import-preview [options]
func (co Controller) OptionsImportYnabImportPreviewV3(c *gin.Context) {
	httputil.OptionsPost(c)
}

// ImportYnabImportPreview parses a YNAB import format CSV and returns a preview of transactions
// to be imported into Envelope Zero.
//
//	@Summary		Transaction Import Preview
//	@Description	Returns a preview of transactions to be imported after parsing a YNAB Import format csv file
//	@Tags			Import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		200			{object}	ImportPreviewListV3
//	@Failure		400			{object}	ImportPreviewListV3
//	@Failure		404			{object}	ImportPreviewListV3
//	@Failure		500			{object}	ImportPreviewListV3
//	@Param			file		formData	file	true	"File to import"
//	@Param			accountId	query		string	false	"ID of the account to import transactions for"
//	@Router			/v3/import/ynab-import-preview [post]
func (co Controller) ImportYnabImportPreviewV3(c *gin.Context) {
	var query ImportPreviewQuery
	err := c.BindQuery(&query)
	// When the binding fails, it is always because the accountID is not set
	if err != nil {
		s := httperrors.ErrAccountIDParameter.Error()
		c.JSON(http.StatusBadRequest, ImportPreviewListV3{
			Error: &s,
		})
		return
	}

	f, e := getUploadedFile(c, ".csv")
	if !e.Nil() {
		s := e.Error()
		c.JSON(e.Status, ImportPreviewListV3{
			Error: &s,
		})
		return
	}

	accountID, e := httputil.UUIDFromString(query.AccountID)
	if !e.Nil() {
		s := e.Error()
		c.JSON(e.Status, ImportPreviewListV3{
			Error: &s,
		})
		return
	}

	// Verify that the account exists
	account, e := getResourceByID[models.Account](c, co, accountID)
	if !e.Nil() {
		s := e.Error()
		c.JSON(e.Status, ImportPreviewListV3{
			Error: &s,
		})
		return
	}

	transactions, err := ynabimport.Parse(f, account)
	if err != nil {
		// ynabimport.Parse parsing returns a usable error already, no parsing necessary
		s := err.Error()
		c.JSON(http.StatusBadRequest, ImportPreviewListV3{
			Error: &s,
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
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, ImportPreviewListV3{
			Error: &s,
		})
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
				e := httperrors.Parse(c, err)
				s := e.Error()
				c.JSON(e.Status, ImportPreviewListV3{
					Error: &s,
				})
				return
			}
		}

		duplicateTransactions(co, &transaction, account.BudgetID)

		// Recommend an envelope
		if transaction.Transaction.DestinationAccountID != uuid.Nil {
			err = recommendEnvelope(co, &transaction, transaction.Transaction.DestinationAccountID)
			if err != nil {
				e := httperrors.Parse(c, err)
				s := e.Error()
				c.JSON(e.Status, ImportPreviewListV3{
					Error: &s,
				})
				return
			}
		}

		transactions[i] = transaction
	}

	c.JSON(http.StatusOK, ImportPreviewListV3{Data: transactions})
}

// ImportYnab4 imports a YNAB 4 budget
//
//	@Summary		Import YNAB 4 budget
//	@Description	Imports budgets from YNAB 4
//	@Tags			Import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		201			{object}	BudgetResponseV3
//	@Failure		400			{object}	BudgetResponseV3
//	@Failure		500			{object}	BudgetResponseV3
//	@Param			file		formData	file	true	"File to import"
//	@Param			budgetName	query		string	false	"Name of the Budget to create"
//	@Router			/v3/import/ynab4 [post]
func (co Controller) ImportYnab4V3(c *gin.Context) {
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
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, ImportPreviewListV3{
			Error: &s,
		})
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

	c.JSON(http.StatusCreated, BudgetResponseV3{Data: &r})
}
