package types

import "github.com/envelope-zero/backend/pkg/models"

// ParsedResources is the struct containing all resources that are to be created
// Named resources are in maps with their names as keys to enable easy deduplication
// and iteration through them.
type ParsedResources struct {
	Budget       models.Budget
	Accounts     map[string]Account
	Categories   map[string]Category
	Allocations  []Allocation
	Transactions []Transaction
}

type Account struct {
	Model models.Account
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

type Transaction struct {
	Model              models.Transaction
	SourceAccount      string
	DestinationAccount string
	Category           string // There is a category here since an envelope with the same name can exist for multiple categories
	Envelope           string
}
