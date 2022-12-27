package ynab4

import (
	"github.com/shopspring/decimal"
	"golang.org/x/text/language"
)

// IDToName is a map of strings to a string.
//
// Use it to map the YNAB 4 Entity IDs to the names
// to enable easier mapping.
type IDToName map[string]string

// IDToEnvelopes maps the ID of a YNAB 4 subcategory to a category and envelope name
// for Envelope Zero.
type IDToEnvelopes map[string]IDToEnvelope

type IDToEnvelope struct {
	Category string
	Envelope string
}

// This has been converted from testdata/Budget.yfull with the help of
// https://mholt.github.io/json-to-go/
//
// Unused fields have been removed to keep the struct as small as possible.
type Budget struct {
	BudgetMetaData struct {
		CurrencyLocale language.Tag `json:"currencyLocale"`
	} `json:"budgetMetaData"`
	Transactions          []Transaction          `json:"transactions"`
	MonthlyBudgets        []MonthlyBudget        `json:"monthlyBudgets"`
	Categories            []Category             `json:"masterCategories"`
	ScheduledTransactions []ScheduledTransaction `json:"scheduledTransactions"`
	Payees                []Payee                `json:"payees"`
	Accounts              []Account              `json:"accounts"`
}

type Account struct {
	EntityID string `json:"entityId"`
	Note     string `json:"note"`
	OnBudget bool   `json:"onBudget"`
	Hidden   bool   `json:"hidden"`
	Name     string `json:"accountName"`
}

type Payee struct {
	EntityID string `json:"entityId"`
	Name     string `json:"name"`
	Deleted  bool   `json:"isTombstone,omitempty"`
}

type SubCategory struct {
	EntityID   string `json:"entityId"`
	CategoryID string `json:"masterCategoryId"`
	Name       string `json:"name"`
	Note       string `json:"note"`
	Deleted    bool   `json:"isTombstone,omitempty"`
}

type Category struct {
	SubCategories []SubCategory `json:"subCategories"`
	EntityID      string        `json:"entityId"`
	Name          string        `json:"name"`
	Note          string        `json:"note"`
	Deleted       bool          `json:"isTombstone,omitempty"`
}

type Transaction struct {
	EntityID        string          `json:"entityId"`
	Amount          decimal.Decimal `json:"amount"`
	CategoryID      string          `json:"categoryId"`
	Date            string          `json:"date"`
	Memo            string          `json:"memo"`
	Deleted         bool            `json:"isTombstone"`
	PayeeID         string          `json:"payeeId"`
	AccountID       string          `json:"accountId"`
	Cleared         string          `json:"cleared"`
	TargetAccountID string          `json:"targetAccountId"`
	SubTransactions []struct {
		CategoryID      string          `json:"categoryId"`
		Amount          decimal.Decimal `json:"amount"`
		Memo            string          `json:"memo"`
		TargetAccountID string          `json:"targetAccountId"`
	} `json:"subTransactions"`
}

type MonthlySubCategoryBudget struct {
	Budgeted             decimal.Decimal `json:"budgeted"`
	OverspendingHandling string          `json:"overspendingHandling"`
	CategoryID           string          `json:"categoryId"`
}

type MonthlyBudget struct {
	Month                     string                     `json:"month"`
	MonthlySubCategoryBudgets []MonthlySubCategoryBudget `json:"monthlySubCategoryBudgets"`
}

// TODO: Clean up when implementing https://github.com/envelope-zero/backend/issues/379
type ScheduledTransaction struct {
	EntityID            string          `json:"entityId"`
	EntityType          string          `json:"entityType"`
	TwiceAMonthStartDay int             `json:"twiceAMonthStartDay"`
	Amount              decimal.Decimal `json:"amount"`
	Frequency           string          `json:"frequency"`
	CategoryID          string          `json:"categoryId"`
	Date                string          `json:"date"`
	Accepted            bool            `json:"accepted"`
	PayeeID             string          `json:"payeeId"`
	EntityVersion       string          `json:"entityVersion"`
	AccountID           string          `json:"accountId"`
	Cleared             string          `json:"cleared"`
	Memo                string          `json:"memo,omitempty"`
}
