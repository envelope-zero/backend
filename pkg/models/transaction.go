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
	Date                 time.Time       `json:"date" example:"1815-12-10T18:43:00.271152Z"`
	Amount               decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)" example:"14.03" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
	Note                 string          `json:"note" example:"Lunch" default:""`
	BudgetID             uuid.UUID       `json:"budgetId" example:"55eecbd8-7c46-4b06-ada9-f287802fb05e"`
	SourceAccountID      uuid.UUID       `json:"sourceAccountId" example:"fd81dc45-a3a2-468e-a6fa-b2618f30aa45"`
	DestinationAccountID uuid.UUID       `json:"destinationAccountId" example:"8e16b456-a719-48ce-9fec-e115cfa7cbcc"`
	EnvelopeID           uuid.UUID       `json:"envelopeId" example:"2649c965-7999-4873-ae16-89d5d5fa972e"`
	Reconciled           bool            `json:"reconciled" example:"true" default:"false"`
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
