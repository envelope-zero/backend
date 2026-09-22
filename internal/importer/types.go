package importer

import (
	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/envelope-zero/backend/v7/internal/types"
	"github.com/google/uuid"
)

// ParsedResources is the struct containing all resources that are to be created
// Named resources are in maps with their names as keys to enable easy deduplication
// and iteration through them.
type ParsedResources struct {
	Budget         models.Budget
	Accounts       []models.Account
	Categories     map[string]Category
	Transactions   []Transaction
	MonthConfigs   []MonthConfig
	MatchRules     []MatchRule
	OverspendFixes []OverspendFix
}

// OverspendFix supports the import of budgeting apps that allow overspending
// for an envelope to affect that envelope's balance in the next month.
// It is used by the creator to subtract the overspent amount from the allocation
// of the next month for the specific envelope
//
// OverspendFixes have to be added by the budget parsers since these are responsible
// for detecting situations where overspend is configured to affect the envelope.
//
// However, the calculation of the balance for the envelope and possible subtraction
// of overspend is handled by the creator
type OverspendFix struct {
	Category string // There is a category here since an envelope with the same name can exist for multiple categories
	Envelope string
	Month    types.Month
}

type Category struct {
	Model     models.Category
	Envelopes map[string]Envelope
}

type Envelope struct {
	Model models.Envelope
}

// MatchRule represents a MatchRule to be imported.
type MatchRule struct {
	models.MatchRule
	Account string
}

const (
	AffectAvailable string = "AFFECT_AVAILABLE"
	AffectEnvelope  string = "AFFECT_ENVELOPE"
)

type MonthConfig struct {
	Model         models.MonthConfig
	Category      string // There is a category here since an envelope with the same name can exist for multiple categories
	Envelope      string
	OverspendMode string // The OverspendMode used by YNAB4
}

type Transaction struct {
	Model                  models.Transaction
	SourceAccountHash      string // Import hash of the source account
	DestinationAccountHash string // Import hash of the destination account
	Category               string // There is a category here since an envelope with the same name can exist for multiple categories
	Envelope               string
}

// TransactionPreview is used to preview transactions that will be imported to allow for editing.
type TransactionPreview struct {
	Transaction             models.Transaction `json:"transaction"`
	SourceAccountName       string             `json:"sourceAccountName" example:"Employer"`                       // Name of the source account from the CSV file
	DestinationAccountName  string             `json:"destinationAccountName" example:"Deutsche Bahn"`             // Name of the destination account from the CSV file
	DuplicateTransactionIDs []uuid.UUID        `json:"duplicateTransactionIds"`                                    // IDs of transactions that this transaction duplicates
	MatchRuleID             uuid.UUID          `json:"matchRuleId" example:"042d101d-f1de-4403-9295-59dc0ea58677"` // ID of the match rule that was applied to this transaction preview
}
