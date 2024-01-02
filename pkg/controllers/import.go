package controllers

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/importer"
	"github.com/envelope-zero/backend/v4/pkg/models"
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

// getUploadedFile returns the form file and handles potential errors.
func getUploadedFile(c *gin.Context, suffix string) (multipart.File, httperrors.Error) {
	formFile, err := c.FormFile("file")
	if formFile == nil {
		return nil, httperrors.Error{
			Status: http.StatusBadRequest,
			Err:    httperrors.ErrNoFilePost,
		}
	}

	if err != nil {
		return nil, httperrors.Parse(c, err)
	}

	if !strings.HasSuffix(formFile.Filename, suffix) {
		return nil, httperrors.Error{
			Status: http.StatusBadRequest,
			Err:    fmt.Errorf("this endpoint only supports %s files", suffix),
		}
	}

	f, err := formFile.Open()
	if err != nil {
		return nil, httperrors.Parse(c, err)
	}

	return f, httperrors.Error{}
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
func findAccounts(co Controller, transaction *importer.TransactionPreview, budgetID uuid.UUID) error {
	// Find the right account name
	name := transaction.DestinationAccountName
	if transaction.SourceAccountName != "" {
		name = transaction.SourceAccountName
	}

	var account models.Account
	err := co.DB.Where(models.Account{
		Name:     name,
		BudgetID: budgetID,
		Archived: false,
	},
		// Account Names are unique, therefore only one can match
		"Name", "BudgetID", "Archived").First(&account).Error

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
	}

	if transaction.DestinationAccountName != "" {
		transaction.Transaction.DestinationAccountID, transaction.MatchRuleID = replace(transaction.DestinationAccountName)
	}
}

// recommendEnvelope sets the first of the recommended envelopes for the opposing account.
func recommendEnvelope(co Controller, transaction *importer.TransactionPreview, id uuid.UUID) error {
	// Load the account
	var destinationAccount models.Account
	err := co.DB.First(&destinationAccount, models.Account{DefaultModel: models.DefaultModel{ID: id}}).Error
	if err != nil {
		return err
	}

	// Preset the most popular recent envelope
	envelopes, err := destinationAccount.RecentEnvelopes(co.DB)
	if err != nil {
		return err
	}

	if len(envelopes) > 0 && envelopes[0] != &uuid.Nil {
		transaction.Transaction.EnvelopeID = envelopes[0]
	}

	return nil
}
