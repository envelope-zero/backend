package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Transaction represents a transaction between two accounts
type Transaction struct {
	Model
	Date                 time.Time       `json:"date"`
	Amount               decimal.Decimal `json:"amount" gorm:"type:DECIMAL(20,8);"`
	Note                 string          `json:"note,omitempty"`
	BudgetID             int             `json:"budgetId"`
	Budget               Budget          `json:"-"`
	SourceAccountID      int             `json:"sourceAccountId"`
	SourceAccount        Account         `json:"-"`
	DestinationAccountID int             `json:"destinationAccountId"`
	DestinationAccount   Account         `json:"-"`
}

// CreateTransaction defines all data needed to create a transaction
type CreateTransaction struct {
	Date                 time.Time       `json:"date" binding:"required"`
	Amount               decimal.Decimal `json:"amount" binding:"required"`
	Note                 string          `json:"note"`
	SourceAccountID      int             `json:"sourceAccountId"`
	DestinationAccountID int             `json:"destinationAccountId"`
}
