package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Transaction represents a transaction between two accounts.
type Transaction struct {
	Model
	TransactionCreate
	Budget             Budget   `json:"-"`
	SourceAccount      Account  `json:"-"`
	DestinationAccount Account  `json:"-"`
	Envelope           Envelope `json:"-"`
}

type TransactionCreate struct {
	Date                 time.Time       `json:"date,omitempty"`
	Amount               decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)"`
	Note                 string          `json:"note,omitempty"`
	BudgetID             uuid.UUID       `json:"budgetId,omitempty"`
	SourceAccountID      uuid.UUID       `json:"sourceAccountId,omitempty"`
	DestinationAccountID uuid.UUID       `json:"destinationAccountId,omitempty"`
	EnvelopeID           uuid.UUID       `json:"envelopeId,omitempty"`
	Reconciled           bool            `json:"reconciled"`
}

// AfterFind updates the timestamps to use UTC as
// timezone, not +0000. Yes, this is different.
//
// We already store them in UTC, but somehow reading
// them from the database returns them as +0000.
func (t *Transaction) AfterFind(tx *gorm.DB) (err error) {
	err = t.Model.AfterFind(tx)
	if err != nil {
		return err
	}

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
