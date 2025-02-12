package v4

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/envelope-zero/backend/v5/internal/httputil"
	"github.com/envelope-zero/backend/v5/internal/importer"
	ynabimport "github.com/envelope-zero/backend/v5/internal/importer/parser/ynab-import"
	"github.com/envelope-zero/backend/v5/internal/importer/parser/ynab4"
	"github.com/envelope-zero/backend/v5/internal/models"
	ez_uuid "github.com/envelope-zero/backend/v5/internal/uuid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ryanuber/go-glob"
)

type ImportQuery struct {
	BudgetName string `form:"budgetName" binding:"required"` // Name for the new budget
}

type ImportPreviewQuery struct {
	AccountID ez_uuid.UUID `form:"accountId" binding:"required"` // ID of the account to import the transactions for
}

// getUploadedFile returns the form file and handles potential errors.
func getUploadedFile(c *gin.Context, suffix string) (multipart.File, error) {
	formFile, err := c.FormFile("file")
	if formFile == nil {
		return nil, errNoFilePost
	}

	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(formFile.Filename, suffix) {
		return nil, fmt.Errorf("%w: %s", errWrongFileSuffix, suffix)
	}

	f, err := formFile.Open()
	if err != nil {
		return nil, err
	}

	return f, nil
}

// duplicateTransactions finds duplicate transactions by their import hash. For all input resources,
// existing resources with the same import hash are searched. If any exist, their IDs are set in the
// DuplicateTransactionIDs field.
func duplicateTransactions(transaction *importer.TransactionPreview, budgetID uuid.UUID) {
	var duplicates []models.Transaction
	models.DB.
		Preload("SourceAccount").
		Preload("DestinationAccount").
		Where(models.Transaction{
			ImportHash: transaction.Transaction.ImportHash,
		}).
		Where(models.Transaction{SourceAccount: models.Account{BudgetID: budgetID}}).
		Or(models.Transaction{DestinationAccount: models.Account{BudgetID: budgetID}}).
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
func findAccounts(transaction *importer.TransactionPreview, budgetID uuid.UUID) error {
	// Find the right account name
	name := transaction.DestinationAccountName
	if transaction.SourceAccountName != "" {
		name = transaction.SourceAccountName
	}

	var account models.Account
	err := models.DB.Where(models.Account{
		Name:     name,
		BudgetID: budgetID,
		Archived: false,
	},
		// Account Names are unique, therefore only one can match
		"Name", "BudgetID", "Archived").First(&account).Error

	// Abort if no accounts are found, but with no error
	// since this is an expected case - there might just
	// not be a matching account
	if errors.Is(err, models.ErrResourceNotFound) {
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
	}

	if transaction.DestinationAccountName != "" {
		transaction.Transaction.DestinationAccountID, transaction.MatchRuleID = replace(transaction.DestinationAccountName)
	}
}

// recommendEnvelope sets the first of the recommended envelopes for the opposing account.
func recommendEnvelope(transaction *importer.TransactionPreview, id uuid.UUID) error {
	// Load the account
	var destinationAccount models.Account
	err := models.DB.First(&destinationAccount, models.Account{DefaultModel: models.DefaultModel{ID: id}}).Error
	if err != nil {
		return err
	}

	// Preset the most popular recent envelope
	envelopes, err := destinationAccount.RecentEnvelopes(models.DB)
	if err != nil {
		return err
	}

	if len(envelopes) > 0 && envelopes[0] != &uuid.Nil {
		transaction.Transaction.EnvelopeID = envelopes[0]
	}

	return nil
}

type ImportPreviewList struct {
	Data  []TransactionPreview `json:"data"`                                                          // List of transaction previews
	Error *string              `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this Match Rule
}

// RegisterImportRoutes registers the routes for imports.
func RegisterImportRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsImport)
		r.GET("", GetImport)

		r.OPTIONS("/ynab4", OptionsImportYnab4)
		r.POST("/ynab4", ImportYnab4)

		r.OPTIONS("/ynab-import-preview", OptionsImportYnabImportPreview)
		r.POST("/ynab-import-preview", ImportYnabImportPreview)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs.
// @Tags			Import
// @Success		204
// @Router			/v4/import [options]
func OptionsImport(c *gin.Context) {
	httputil.OptionsGet(c)
}

type ImportResponse struct {
	Links ImportLinks `json:"links"` // Links for the v4 API
}

type ImportLinks struct {
	Ynab4             string `json:"transactions" example:"https://example.com/api/v4/import/ynab4"`             // URL of YNAB4 import endpoint
	YnabImportPreview string `json:"matchRules" example:"https://example.com/api/v4/import/ynab-import-preview"` // URL of YNAB Import preview endpoint
}

// @Summary		Import API overview
// @Description	Returns general information about the v4 API
// @Tags			Import
// @Success		200	{object}	ImportResponse
// @Router			/v4/import [get]
func GetImport(c *gin.Context) {
	c.JSON(http.StatusOK, ImportResponse{
		Links: ImportLinks{
			Ynab4:             c.GetString(string(models.DBContextURL)) + "/v4/import/ynab4",
			YnabImportPreview: c.GetString(string(models.DBContextURL)) + "/v4/import/ynab-import-preview",
		},
	})
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Import
// @Success		204
// @Router			/v4/import/ynab4 [options]
func OptionsImportYnab4(c *gin.Context) {
	httputil.OptionsPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Import
// @Success		204
// @Router			/v4/import/ynab-import-preview [options]
func OptionsImportYnabImportPreview(c *gin.Context) {
	httputil.OptionsPost(c)
}

// @Summary		Transaction Import Preview
// @Description	Returns a preview of transactions to be imported after parsing a YNAB Import format csv file
// @Tags			Import
// @Accept			multipart/form-data
// @Produce		json
// @Success		200			{object}	ImportPreviewList
// @Failure		400			{object}	ImportPreviewList
// @Failure		404			{object}	ImportPreviewList
// @Failure		500			{object}	ImportPreviewList
// @Param			file		formData	file				true	"File to import"
// @Param			accountId	query		ImportPreviewQuery	false	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/import/ynab-import-preview [post]
func ImportYnabImportPreview(c *gin.Context) {
	var query ImportPreviewQuery
	err := c.BindQuery(&query)
	if err != nil {
		s := fmt.Errorf("accountId: %w", err).Error()
		c.JSON(http.StatusBadRequest, ImportPreviewList{
			Error: &s,
		})
		return
	}

	if query.AccountID == ez_uuid.Nil {
		s := errAccountIDParameter.Error()
		c.JSON(http.StatusBadRequest, ImportPreviewList{
			Error: &s,
		})
		return
	}

	f, err := getUploadedFile(c, ".csv")
	if err != nil {
		s := err.Error()
		c.JSON(status(err), ImportPreviewList{
			Error: &s,
		})
		return
	}

	// Verify that the account exists
	var account models.Account
	err = models.DB.First(&account, query.AccountID).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), ImportPreviewList{
			Error: &s,
		})
		return
	}

	transactions, err := ynabimport.Parse(f, account)
	if err != nil {
		// ynabimport.Parse parsing returns a usable error already, no parsing necessary
		s := err.Error()
		c.JSON(http.StatusBadRequest, ImportPreviewList{
			Error: &s,
		})
		return
	}

	// Get all match rules for the budget for which the account is not archived
	var matchRules []models.MatchRule
	err = models.DB.
		Joins("JOIN accounts ON accounts.budget_id = ? AND NOT accounts.archived", account.BudgetID).
		Joins("JOIN match_rules rr ON rr.account_id = accounts.id").
		Order("rr.priority asc").
		Find(&matchRules).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), ImportPreviewList{
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
			err = findAccounts(&transaction, account.BudgetID)
			if err != nil {
				s := err.Error()
				c.JSON(status(err), ImportPreviewList{
					Error: &s,
				})
				return
			}
		}

		duplicateTransactions(&transaction, account.BudgetID)

		// Recommend an envelope
		if transaction.Transaction.DestinationAccountID != uuid.Nil {
			err = recommendEnvelope(&transaction, transaction.Transaction.DestinationAccountID)
			if err != nil {
				s := err.Error()
				c.JSON(status(err), ImportPreviewList{
					Error: &s,
				})
				return
			}
		}

		transactions[i] = transaction
	}

	// We need to transform the responses for v4
	data := make([]TransactionPreview, 0, len(transactions))
	for _, t := range transactions {
		data = append(data, newTransactionPreview(c, t))
	}

	c.JSON(http.StatusOK, ImportPreviewList{Data: data})
}

// @Summary		Import YNAB 4 budget
// @Description	Imports budgets from YNAB 4
// @Tags			Import
// @Accept			multipart/form-data
// @Produce		json
// @Success		201			{object}	BudgetResponse
// @Failure		400			{object}	BudgetResponse
// @Failure		500			{object}	BudgetResponse
// @Param			file		formData	file		true	"File to import"
// @Param			budgetName	query		ImportQuery	false	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/import/ynab4 [post]
func ImportYnab4(c *gin.Context) {
	var query ImportQuery
	if err := c.BindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, httpError{Error: errBudgetNameNotSet.Error()})
		return
	}

	// Verify if the budget does already exist. If yes, return an error
	// as we only allow imports to new budgets
	var budget models.Budget
	err := models.DB.Where(&models.Budget{
		Name: query.BudgetName,
	}).First(&budget).Error

	if err == nil {
		c.JSON(http.StatusBadRequest, httpError{
			Error: errBudgetNameInUse.Error(),
		})
		return
	} else if err != nil && !errors.Is(err, models.ErrResourceNotFound) {
		s := err.Error()
		c.JSON(status(err), BudgetResponse{
			Error: &s,
		})
		return
	}

	f, err := getUploadedFile(c, ".yfull")
	if err != nil {
		s := err.Error()
		c.JSON(status(err), BudgetResponse{
			Error: &s,
		})
		return
	}

	// Parse the Budget.yfull
	resources, err := ynab4.Parse(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpError{Error: err.Error()})
		return
	}

	// Set the budget name explicitly since YNAB 4 files
	// do not contain it
	resources.Budget.Name = query.BudgetName

	budget, err = importer.Create(models.DB, resources)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), BudgetResponse{
			Error: &s,
		})
		return
	}

	data := newBudget(c, budget)
	c.JSON(http.StatusCreated, BudgetResponse{Data: &data})
}
