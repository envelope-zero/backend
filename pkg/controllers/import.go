package controllers

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/importer"
	ynabimport "github.com/envelope-zero/backend/v3/pkg/importer/parser/ynab-import"
	"github.com/envelope-zero/backend/v3/pkg/importer/parser/ynab4"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ryanuber/go-glob"
	"gorm.io/gorm"
)

type ImportQuery struct {
	BudgetName string `form:"budgetName" binding:"required"` // Name for the new budget
}

type ImportPreviewQuery struct {
	AccountID string `form:"accountId" binding:"required"` // ID of the account to import the transactions for
}

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
func duplicateTransactions(co Controller, transaction *importer.TransactionPreview, budgetID uuid.UUID) {
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
}

// findAccounts sets the source or destination account ID for a TransactionPreview resource
// if there is exactly one account with a matching name.
func findAccounts(co Controller, transaction *importer.TransactionPreview, budgetID uuid.UUID) error {
	// Find the right account name
	name := transaction.DestinationAccountName
	if transaction.SourceAccountName != "" {
		name = transaction.SourceAccountName
	}

	var account models.Account
	err := co.DB.Where(models.Account{
		AccountCreate: models.AccountCreate{
			Name:     name,
			BudgetID: budgetID,
			Hidden:   false,
		},
	},
		// Account Names are unique, therefore only one can match
		"Name", "BudgetID", "Hidden").First(&account).Error

	// Abort if no accounts are found, but with no error
	// since this is an expected case - there might just
	// not be a matching account
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	// Set source or destination, depending on which one we checked for
	if account.ID != uuid.Nil {
		if transaction.SourceAccountName != "" {
			transaction.Transaction.SourceAccountID = account.ID
		} else {
			transaction.Transaction.DestinationAccountID = account.ID
		}
	}

	return nil
}

// match applies the match rules to a transaction.
func match(transaction *importer.TransactionPreview, rules []models.MatchRule) {
	replace := func(name string) (uuid.UUID, uuid.UUID) {
		// Iterate over all rules
		for _, rule := range rules {
			// If the rule matches, return the account ID. Since rules are loaded from
			// the database in priority order, we can simply return the first match
			if glob.Glob(rule.Match, name) {
				return rule.AccountID, rule.ID
			}
		}
		return uuid.Nil, uuid.Nil
	}

	if transaction.SourceAccountName != "" {
		transaction.Transaction.SourceAccountID, transaction.MatchRuleID = replace(transaction.SourceAccountName)

		// This is kept for backwards compatibility and will be removed with API version 3
		// https://github.com/envelope-zero/backend/issues/763
		transaction.RenameRuleID = transaction.MatchRuleID
	}

	if transaction.DestinationAccountName != "" {
		transaction.Transaction.DestinationAccountID, transaction.MatchRuleID = replace(transaction.DestinationAccountName)

		// This is kept for backwards compatibility and will be removed with API version 3
		// https://github.com/envelope-zero/backend/issues/763
		transaction.RenameRuleID = transaction.MatchRuleID
	}
}

// recommendEnvelope sets the first of the recommended envelopes for the opposing account.
func recommendEnvelope(co Controller, transaction *importer.TransactionPreview, id uuid.UUID) error {
	// Load the account
	var destinationAccount models.AccountV2
	err := co.DB.First(&destinationAccount, models.AccountV2{DefaultModel: models.DefaultModel{ID: id}}).Error
	if err != nil {
		return err
	}

	// Preset the most popular recent envelope
	err = destinationAccount.SetRecentEnvelopes(co.DB)
	if err != nil {
		return err
	}

	if len(destinationAccount.RecentEnvelopes) > 0 && destinationAccount.RecentEnvelopes[0] != &uuid.Nil {
		transaction.Transaction.EnvelopeID = destinationAccount.RecentEnvelopes[0]
	}

	return nil
}
