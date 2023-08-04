package controllers

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/envelope-zero/backend/v2/pkg/httputil"
	"github.com/envelope-zero/backend/v2/pkg/importer"
	ynabimport "github.com/envelope-zero/backend/v2/pkg/importer/parser/ynab-import"
	"github.com/envelope-zero/backend/v2/pkg/importer/parser/ynab4"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ImportQuery struct {
	BudgetName string `form:"budgetName" binding:"required"`
}

type ImportPreviewQuery struct {
	AccountID string `form:"accountId" binding:"required"`
}

type ImportPreviewList struct {
	Data []importer.TransactionPreview `json:"data"`
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
//	@Failure		500	{object}	httperrors.HTTPError
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
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/import/ynab4 [options]
func (co Controller) OptionsImportYnab4(c *gin.Context) {
	httputil.OptionsPost(c)
}

// OptionsImportYnab4 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Import
//	@Success		204
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/import/ynab-import-preview [options]
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
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			file		formData	file	true	"File to import"
//	@Param			accountId	query		string	false	"ID of the account to import transactions for"
//	@Router			/v1/import/ynab-import-preview [post]
func (co Controller) ImportYnabImportPreview(c *gin.Context) {
	var query ImportPreviewQuery
	if err := c.BindQuery(&query); err != nil {
		httperrors.New(c, http.StatusBadRequest, "The accountId parameter must be set")
		return
	}

	f, ok := getUploadedFile(c, ".csv")
	if !ok {
		return
	}

	accountID, ok := httputil.UUIDFromString(c, query.AccountID)
	if !ok {
		return
	}

	// Verify that the account exists
	account, ok := getResourceByIDAndHandleErrors[models.Account](c, co, accountID)
	if !ok {
		return
	}

	transactions, err := ynabimport.Parse(f, account)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for i, transaction := range transactions {
		transaction, err = findAccounts(co, transaction, account.BudgetID)
		transaction = duplicateTransactions(co, transaction, account.BudgetID)
		transactions[i] = transaction
	}

	if err != nil {
		httperrors.Handler(c, err)
		return
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

	f, ok := getUploadedFile(c, ".yfull")
	if !ok {
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

	c.JSON(http.StatusCreated, BudgetResponse{Data: budget})
}

// getUploadedFile returns the form file and handles potential errors.
func getUploadedFile(c *gin.Context, suffix string) (multipart.File, bool) {
	formFile, err := c.FormFile("file")
	if formFile == nil {
		httperrors.New(c, http.StatusBadRequest, "You must send a file to this endpoint")
		return nil, false
	} else if err != nil && err.Error() == "unexpected EOF" {
		httperrors.New(c, http.StatusBadRequest, "The file you uploaded is empty. Did the file get deleted before you uploaded it?")
		return nil, false
	} else if err != nil {
		httperrors.Handler(c, err)
		return nil, false
	}

	if !strings.HasSuffix(formFile.Filename, suffix) {
		httperrors.New(c, http.StatusBadRequest, fmt.Sprintf("This endpoint only supports %s files", suffix))
		return nil, false
	}

	f, err := formFile.Open()
	if err != nil {
		httperrors.Handler(c, err)
		return nil, false
	}

	return f, true
}

// duplicateTransactions finds duplicate transactions by their import hash. For all input resources,
// existing resources with the same import hash are searched. If any exist, their IDs are set in the
// DuplicateTransactionIDs field.
func duplicateTransactions(co Controller, transaction importer.TransactionPreview, budgetID uuid.UUID) importer.TransactionPreview {
	var duplicates []models.Transaction
	co.DB.
		Preload("SourceAccount").
		Preload("DestinationAccount").
		Where(models.Transaction{
			TransactionCreate: models.TransactionCreate{
				ImportHash: transaction.Transaction.ImportHash,
			},
		}).
		Where(models.Transaction{SourceAccount: models.Account{AccountCreate: models.AccountCreate{BudgetID: budgetID}}}).
		Or(models.Transaction{DestinationAccount: models.Account{AccountCreate: models.AccountCreate{BudgetID: budgetID}}}).
		Find(&duplicates)

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	duplicateIDs := make([]uuid.UUID, 0)
	for _, duplicate := range duplicates {
		if duplicate.SourceAccount.BudgetID == budgetID || duplicate.DestinationAccount.BudgetID == budgetID {
			duplicateIDs = append(duplicateIDs, duplicate.ID)
		}
	}
	transaction.DuplicateTransactionIDs = duplicateIDs

	return transaction
}

// findAccounts sets the source or destination account ID for a TransactionPreview resource
// if there is exactly one account with a matching name.
func findAccounts(co Controller, transaction importer.TransactionPreview, budgetID uuid.UUID) (importer.TransactionPreview, error) {
	// Find the right account name
	name := transaction.DestinationAccountName
	if transaction.SourceAccountName != "" {
		name = transaction.SourceAccountName
	}

	var accounts []models.Account
	co.DB.Where(models.Account{
		AccountCreate: models.AccountCreate{
			Name:     name,
			BudgetID: budgetID,
			Hidden:   false,
		},
	},
		// Explicitly specfiy search fields since we use a zero value for Hidden
		"Name", "BudgetID", "Hidden").Find(&accounts)

	// We cannot determine correctly which account should be used if there are
	// multiple accounts, therefore we skip
	//
	// We also continue if no accounts are found
	if len(accounts) != 1 {
		return transaction, nil
	}

	// Set source or destination, depending on which one we checked for
	if accounts[0].ID != uuid.Nil {
		if transaction.SourceAccountName != "" {
			transaction.Transaction.SourceAccountID = accounts[0].ID
			transaction.SourceAccountName = ""
		} else {
			transaction.Transaction.DestinationAccountID = accounts[0].ID
			transaction.DestinationAccountName = ""
		}

		// Preset the most popular recent envelope
		recentEnvelopes, err := accounts[0].RecentEnvelopes(co.DB)
		if err != nil {
			return importer.TransactionPreview{}, err
		}

		if len(recentEnvelopes) > 0 && recentEnvelopes[0].ID != uuid.Nil {
			transaction.Transaction.EnvelopeID = &recentEnvelopes[0].ID
		}
	}

	return transaction, nil
}
