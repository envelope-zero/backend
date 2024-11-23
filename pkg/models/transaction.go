package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Transaction represents a transaction between two accounts.
type Transaction struct {
	DefaultModel
	SourceAccountID       uuid.UUID `gorm:"check:source_destination_different,source_account_id != destination_account_id"`
	SourceAccount         Account   `json:"-"`
	DestinationAccountID  uuid.UUID
	DestinationAccount    Account `json:"-"`
	EnvelopeID            *uuid.UUID
	Envelope              Envelope        `json:"-"`
	Date                  time.Time       // Time of day is currently only used for sorting
	Amount                decimal.Decimal `gorm:"type:DECIMAL(20,8)"`
	Note                  string
	ReconciledSource      bool        // Is the transaction reconciled in the source account?
	ReconciledDestination bool        // Is the transaction reconciled in the destination account?
	AvailableFrom         types.Month // Only used for income transactions. Defaults to the transaction date.
	ImportHash            string      // The SHA256 hash of a unique combination of values to use in duplicate detection when importing transactions
}

var (
	ErrAvailabilityMonthTooEarly                      = errors.New("availability month must not be earlier than the month of the transaction")
	ErrSourceDoesNotEqualDestination                  = errors.New("source and destination accounts for a transaction must be different")
	ErrTransactionAmountNotPositive                   = errors.New("the transaction amount must be positive")
	ErrTransactionNoInternalAccounts                  = errors.New("a transaction between two external accounts is not possible")
	ErrTransactionTransferBetweenOnBudgetWithEnvelope = errors.New("transfers between two on-budget accounts must not have an envelope set. Such a transaction would be incoming and outgoing for this envelope at the same time, which is not possible")
	ErrTransactionInvalidSourceAccount                = errors.New("invalid source account")
	ErrTransactionInvalidDestinationAccount           = errors.New("invalid destination account")
)

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	_ = t.DefaultModel.BeforeCreate(tx)

	toSave := tx.Statement.Dest.(*Transaction)

	if !decimal.Decimal.IsPositive(toSave.Amount) {
		return ErrTransactionAmountNotPositive
	}

	var source Account
	err := tx.First(&source, toSave.SourceAccountID).Error
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransactionInvalidSourceAccount, err)
	}

	var destination Account
	err = tx.First(&destination, toSave.DestinationAccountID).Error
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransactionInvalidDestinationAccount, err)
	}

	return t.checkIntegrity(tx, *toSave, source, destination)
}

func (t *Transaction) BeforeUpdate(tx *gorm.DB) (err error) {
	toSave := tx.Statement.Dest.(Transaction)

	if tx.Statement.Changed("Amount") && !decimal.Decimal.IsPositive(toSave.Amount) {
		return ErrTransactionAmountNotPositive
	}

	var sourceAccountID uuid.UUID
	if tx.Statement.Changed("SourceAccountID") {
		sourceAccountID = toSave.SourceAccountID
	} else {
		sourceAccountID = t.SourceAccountID
	}
	var source Account
	err = tx.First(&source, sourceAccountID).Error
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransactionInvalidSourceAccount, err)
	}

	var destinationAccountID uuid.UUID
	if tx.Statement.Changed("DestinationAccountID") {
		destinationAccountID = toSave.DestinationAccountID
	} else {
		destinationAccountID = t.DestinationAccountID
	}
	var destination Account
	err = tx.First(&destination, destinationAccountID).Error
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransactionInvalidDestinationAccount, err)
	}

	return t.checkIntegrity(tx, toSave, source, destination)
}

func (t *Transaction) checkIntegrity(tx *gorm.DB, toSave Transaction, source, destination Account) error {
	if source.External && destination.External {
		return ErrTransactionNoInternalAccounts
	}

	// Check envelope being set for transfer between on-budget accounts
	if toSave.EnvelopeID != nil && *toSave.EnvelopeID != uuid.Nil {
		if source.OnBudget && destination.OnBudget {
			return ErrTransactionTransferBetweenOnBudgetWithEnvelope
		}
		err := tx.First(&Envelope{}, *toSave.EnvelopeID).Error
		return err
	}

	return nil
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
		t.AvailableFrom = types.MonthOf(t.Date).AddDate(0, 1)
	} else if t.AvailableFrom.Before(types.MonthOf(t.Date)) {
		return fmt.Errorf("%w, transaction date: %s, available month %s", ErrAvailabilityMonthTooEarly, t.Date.Format("2006-01-02"), t.AvailableFrom)
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

// Returns all transactions on this instance for export
func (Transaction) Export() (json.RawMessage, error) {
	var transactions []Transaction
	err := DB.Unscoped().Where(&Transaction{}).Find(&transactions).Error
	if err != nil {
		return nil, err
	}

	j, err := json.Marshal(&transactions)
	if err != nil {
		return json.RawMessage{}, err
	}
	return json.RawMessage(j), nil
}
