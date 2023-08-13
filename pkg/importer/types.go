package importer

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/google/uuid"
)

// ParsedResources is the struct containing all resources that are to be created
// Named resources are in maps with their names as keys to enable easy deduplication
// and iteration through them.
type ParsedResources struct {
	Budget       models.Budget
	Accounts     []models.Account
	Categories   map[string]Category
	Allocations  []Allocation
	Transactions []Transaction
	MonthConfigs []MonthConfig
}

type Category struct {
	Model     models.Category
	Envelopes map[string]Envelope
}

type Envelope struct {
	Model models.Envelope
}

type Allocation struct {
	Model    models.Allocation
	Category string // There is a category here since an envelope with the same name can exist for multiple categories
	Envelope string
}

type MonthConfig struct {
	Model    models.MonthConfig
	Category string // There is a category here since an envelope with the same name can exist for multiple categories
	Envelope string
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
	Transaction             models.TransactionCreate `json:"transaction"`
	SourceAccountName       string                   `json:"sourceAccountName" example:"Employer"`                        // Name of the source account from the CSV file
	DestinationAccountName  string                   `json:"destinationAccountName" example:"Deutsche Bahn"`              // Name of the destination account from the CSV file
	DuplicateTransactionIDs []uuid.UUID              `json:"duplicateTransactionIds"`                                     // IDs of transactions that this transaction duplicates
	RenameRuleID            uuid.UUID                `json:"renameRuleId" example:"042d101d-f1de-4403-9295-59dc0ea58677"` // ID of the rename rule that was applied to this transaction preview
}
