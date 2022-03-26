package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Transaction represents a transaction between two accounts.
type Transaction struct {
	Model
	Date                 *time.Time      `json:"date,omitempty"`
	Amount               decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)"`
	Note                 string          `json:"note,omitempty"`
	BudgetID             int             `json:"budgetId,omitempty"`
	Budget               Budget          `json:"-"`
	SourceAccountID      int             `json:"sourceAccountId,omitempty"`
	SourceAccount        Account         `json:"-"`
	DestinationAccountID int             `json:"destinationAccountId,omitempty"`
	DestinationAccount   Account         `json:"-"`
	EnvelopeID           int             `json:"envelopeId,omitempty"`
	Envelope             Envelope        `json:"-"`
	Reconciled           bool            `json:"reconciled"`
}
