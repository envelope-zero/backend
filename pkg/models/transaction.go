package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Transaction represents a transaction between two accounts.
type Transaction struct {
	DefaultModel
	BudgetID              uuid.UUID
	Budget                Budget
	SourceAccountID       uuid.UUID `gorm:"check:source_destination_different,source_account_id != destination_account_id"`
	SourceAccount         Account
	DestinationAccountID  uuid.UUID
	DestinationAccount    Account
	EnvelopeID            *uuid.UUID
	Envelope              Envelope
	Date                  time.Time       // Time of day is currently only used for sorting
	Amount                decimal.Decimal `gorm:"type:DECIMAL(20,8)"`
	Note                  string
	ReconciledSource      bool        // Is the transaction reconciled in the source account?
	ReconciledDestination bool        // Is the transaction reconciled in the destination account?
	AvailableFrom         types.Month // Only used for income transactions. Defaults to the transaction date.
	ImportHash            string      // The SHA256 hash of a unique combination of values to use in duplicate detection when importing transactions
}

func (t Transaction) Self() string {
	return "Transaction"
}

// BeforeSave
//   - ensures that ReconciledSource and ReconciledDestination are set to valid values
//   - trims whitespace from string fields
func (t *Transaction) BeforeSave(tx *gorm.DB) (err error) {
	t.Note = strings.TrimSpace(t.Note)
	t.ImportHash = strings.TrimSpace(t.ImportHash)

	// Ensure that the Envelope ID is nil and not a pointer to a nil UUID
	// when it is set
	if t.EnvelopeID != nil && *t.EnvelopeID == uuid.Nil {
		t.EnvelopeID = nil
	}

	if t.Date.IsZero() {
		t.Date = time.Now()
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
