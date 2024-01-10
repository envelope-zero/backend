package v4

import (
	"github.com/envelope-zero/backend/v4/pkg/importer"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// newTransactionPreview transforms a TransactionPreview to the API resource
func newTransactionPreview(c *gin.Context, t importer.TransactionPreview) TransactionPreview {
	id := &t.MatchRuleID
	if t.MatchRuleID == uuid.Nil {
		id = nil
	}

	return TransactionPreview{
		Transaction:             newTransaction(c, t.Transaction),
		SourceAccountName:       t.SourceAccountName,
		DestinationAccountName:  t.DestinationAccountName,
		DuplicateTransactionIDs: t.DuplicateTransactionIDs,
		MatchRuleID:             id,
	}
}

// TransactionPreview is used to preview transactions that will be imported to allow for editing.
type TransactionPreview struct {
	Transaction             Transaction `json:"transaction"`
	SourceAccountName       string      `json:"sourceAccountName" example:"Employer"`                       // Name of the source account from the CSV file
	DestinationAccountName  string      `json:"destinationAccountName" example:"Deutsche Bahn"`             // Name of the destination account from the CSV file
	DuplicateTransactionIDs []uuid.UUID `json:"duplicateTransactionIds"`                                    // IDs of transactions that this transaction duplicates
	MatchRuleID             *uuid.UUID  `json:"matchRuleId" example:"042d101d-f1de-4403-9295-59dc0ea58677"` // ID of the match rule that was applied to this transaction preview
}
