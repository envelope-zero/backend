package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Transaction represents a transaction between two accounts.
type Transaction struct {
	Model
	Date                 time.Time       `json:"date,omitempty"`
	Amount               decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)"`
	Note                 string          `json:"note,omitempty"`
	BudgetID             uint64          `json:"budgetId,omitempty"`
	Budget               Budget          `json:"-"`
	SourceAccountID      uint64          `json:"sourceAccountId,omitempty"`
	SourceAccount        Account         `json:"-"`
	DestinationAccountID uint64          `json:"destinationAccountId,omitempty"`
	DestinationAccount   Account         `json:"-"`
	EnvelopeID           uint64          `json:"envelopeId,omitempty"`
	Envelope             Envelope        `json:"-"`
	Reconciled           bool            `json:"reconciled"`
}

// AfterFind updates the timestamps to use UTC as
// timezone, not +0000. Yes, this is different.
//
// We already store them in UTC, but somehow reading
// them from the database returns them as +0000.
func (t *Transaction) AfterFind(tx *gorm.DB) (err error) {
	t.Date = t.Date.In(time.UTC)
	return nil
}

// BeforeSave sets the timezone for the Date for UTC.
func (t *Transaction) BeforeSave(tx *gorm.DB) (err error) {
	if t.Date.IsZero() {
		t.Date = time.Now().In(time.UTC)
	} else {
		t.Date = t.Date.In(time.UTC)
	}

	return nil
}
