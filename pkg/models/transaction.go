package models

import (
	"fmt"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Transaction represents a transaction between two accounts.
type Transaction struct {
	DefaultModel
	TransactionCreate
	Budget             Budget   `json:"-"`
	SourceAccount      Account  `json:"-"`
	DestinationAccount Account  `json:"-"`
	Envelope           Envelope `json:"-"`
}

type TransactionCreate struct {
	Date time.Time `json:"date" example:"1815-12-10T18:43:00.271152Z"` // Date of the transaction. Time is currently only used for sorting
	// The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
	Amount                decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8)" example:"14.03" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"`                        // The amount for the transaction
	Note                  string          `json:"note" example:"Lunch" default:""`                                                                                                                      // A note
	BudgetID              uuid.UUID       `json:"budgetId" example:"55eecbd8-7c46-4b06-ada9-f287802fb05e"`                                                                                              // ID of the budget
	SourceAccountID       uuid.UUID       `json:"sourceAccountId" gorm:"check:source_destination_different,source_account_id != destination_account_id" example:"fd81dc45-a3a2-468e-a6fa-b2618f30aa45"` // ID of the source account
	DestinationAccountID  uuid.UUID       `json:"destinationAccountId" example:"8e16b456-a719-48ce-9fec-e115cfa7cbcc"`                                                                                  // ID of the destination account
	EnvelopeID            *uuid.UUID      `json:"envelopeId" example:"2649c965-7999-4873-ae16-89d5d5fa972e"`                                                                                            // ID of the envelope
	Reconciled            bool            `json:"reconciled" example:"true" default:"false"`                                                                                                            // DEPRECATED. Do not use, this field does not work as intended. See https://github.com/envelope-zero/backend/issues/528. Use reconciledSource and reconciledDestination instead.
	ReconciledSource      bool            `json:"reconciledSource" example:"true" default:"false"`                                                                                                      // Is the transaction reconciled in the source account?
	ReconciledDestination bool            `json:"reconciledDestination" example:"true" default:"false"`                                                                                                 // Is the transaction reconciled in the destination account?

	AvailableFrom types.Month `json:"availableFrom" example:"2021-11-17T00:00:00Z"` // The date from which on the transaction amount is available for budgeting. Only used for income transactions. Defaults to the transaction date.

	ImportHash string `json:"importHash" example:"867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70" default:""` // The SHA256 hash of a unique combination of values to use in duplicate detection
}

func (t Transaction) Self() string {
	return "Transaction"
}

// AfterFind updates the timestamps to use UTC as
// timezone, not +0000. Yes, this is different.
//
// We already store them in UTC, but somehow reading
// them from the database returns them as +0000.
func (t *Transaction) AfterFind(tx *gorm.DB) (err error) {
	err = t.DefaultModel.AfterFind(tx)
	if err != nil {
		return err
	}

	// Enforce dates to be in UTC
	t.Date = t.Date.In(time.UTC)
	return
}

// BeforeSave
//   - sets the timezone for the Date for UTC
//   - ensures that ReconciledSource and ReconciledDestination are set to valid values
func (t *Transaction) BeforeSave(tx *gorm.DB) (err error) {
	// Ensure that the Envelope ID is nil and not a pointer to a nil UUID
	// when it is set
	if t.EnvelopeID != nil && *t.EnvelopeID == uuid.Nil {
		t.EnvelopeID = nil
	}

	if t.Date.IsZero() {
		t.Date = time.Now().In(time.UTC)
	} else {
		t.Date = t.Date.In(time.UTC)
	}

	// Default the AvailableForBudget date to the transaction date
	if t.AvailableFrom.IsZero() {
		t.AvailableFrom = types.MonthOf(t.Date)
	} else if t.AvailableFrom.Before(types.MonthOf(t.Date)) {
		return fmt.Errorf("availability month must not be earlier than the month of the transaction, transaction date: %s, available month %s", t.Date.Format("2006-01-02"), t.AvailableFrom)
	}

	// Enforce ReconciledSource = false when source account is external
	// Only verify when ReconciledSource is true as false is always acceptable
	if t.SourceAccount.ID == uuid.Nil && t.ReconciledSource {
		a := Account{}
		err = tx.Where(&Account{DefaultModel: DefaultModel{ID: t.SourceAccountID}}).First(&a).Error
		if err != nil {
			return fmt.Errorf("no existing account with specified SourceAccountID: %w", err)
		}

		if a.External {
			t.ReconciledSource = false
		}

		// We only need to enforce the value if the source account is external,
		// therefore else if is acceptable here
	} else if t.SourceAccount.External {
		t.ReconciledSource = false
	}

	// Enforce ReconciledDestination = false when destination account is external
	// Only verify when ReconciledDestination is true as false is always acceptable
	if t.DestinationAccount.ID == uuid.Nil && t.ReconciledDestination {
		a := Account{}
		err = tx.Where(&Account{DefaultModel: DefaultModel{ID: t.DestinationAccountID}}).First(&a).Error
		if err != nil {
			return fmt.Errorf("no existing account with specified DestinationAccountID: %w", err)
		}

		if a.External {
			t.ReconciledDestination = false
		}

		// We only need to enforce the value if the source account is external,
		// therefore else if is acceptable here
	} else if t.DestinationAccount.External {
		t.ReconciledDestination = false
	}

	return
}
