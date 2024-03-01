package v4

import (
	"fmt"
	"time"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountEditable struct {
	Name               string          `json:"name" example:"Cash" default:""`                                                                                           // Name of the account
	Note               string          `json:"note" example:"Money in my wallet" default:""`                                                                             // A longer description for the account
	BudgetID           uuid.UUID       `json:"budgetId" example:"550dc009-cea6-4c12-b2a5-03446eb7b7cf"`                                                                  // ID of the budget this account belongs to
	OnBudget           bool            `json:"onBudget" example:"true" default:"false"`                                                                                  // Does the account factor into the available budget? Always false when external: true
	External           bool            `json:"external" example:"false" default:"false"`                                                                                 // Does the account belong to the budget owner or not?
	InitialBalance     decimal.Decimal `json:"initialBalance" example:"173.12" default:"0" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // Balance of the account before any transactions were recorded
	InitialBalanceDate *time.Time      `json:"initialBalanceDate" example:"2017-05-12T00:00:00Z"`                                                                        // Date of the initial balance
	Archived           bool            `json:"archived" example:"true" default:"false"`                                                                                  // Is the account archived?
	ImportHash         string          `json:"importHash" example:"867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70" default:""`                         // The SHA256 hash of a unique combination of values to use in duplicate detection for imports
}

// model returns the database resource for the editable fields
func (editable AccountEditable) model() models.Account {
	return models.Account{
		Name:               editable.Name,
		Note:               editable.Note,
		BudgetID:           editable.BudgetID,
		OnBudget:           editable.OnBudget,
		External:           editable.External,
		InitialBalance:     editable.InitialBalance,
		InitialBalanceDate: editable.InitialBalanceDate,
		Archived:           editable.Archived,
		ImportHash:         editable.ImportHash,
	}
}

type AccountLinks struct {
	Self            string `json:"self" example:"https://example.com/api/v4/accounts/af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`                             // The account itself
	RecentEnvelopes string `json:"recentEnvelopes" example:"https://example.com/api/v4/accounts/af892e10-7e0a-4fb8-b1bc-4b6d88401ed2/recent-envelopes"` // Envelopes in recent transactions where this account was the target
	ComputedData    string `json:"computedData" example:"https://example.com/api/v4/accounts/computed"`                                                 // Computed data endpoint for accounts
	Transactions    string `json:"transactions" example:"https://example.com/api/v4/transactions?account=af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`         // Transactions referencing the account
}

// Account is the API v4 representation of an Account in EZ.
type Account struct {
	models.DefaultModel
	AccountEditable
	Links AccountLinks `json:"links"`
}

func newAccount(c *gin.Context, model models.Account) Account {
	url := c.GetString(string(models.DBContextURL))

	return Account{
		DefaultModel: model.DefaultModel,
		AccountEditable: AccountEditable{
			Name:               model.Name,
			Note:               model.Note,
			BudgetID:           model.BudgetID,
			OnBudget:           model.OnBudget,
			External:           model.External,
			InitialBalance:     model.InitialBalance,
			InitialBalanceDate: model.InitialBalanceDate,
			Archived:           model.Archived,
			ImportHash:         model.ImportHash,
		},
		Links: AccountLinks{
			Self:            fmt.Sprintf("%s/v4/accounts/%s", url, model.ID),
			RecentEnvelopes: fmt.Sprintf("%s/v4/accounts/%s/recent-envelopes", url, model.ID),
			ComputedData:    fmt.Sprintf("%s/v4/accounts/computed", url),
			Transactions:    fmt.Sprintf("%s/v4/transactions?account=%s", url, model.ID),
		},
	}
}

type AccountListResponse struct {
	Data       []Account   `json:"data"`                                                          // List of accounts
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type AccountCreateResponse struct {
	Error *string           `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []AccountResponse `json:"data"`                                                          // List of created Accounts
}

func (a *AccountCreateResponse) appendError(err error, currentStatus int) int {
	s := err.Error()
	a.Data = append(a.Data, AccountResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	newStatus := status(err)
	if newStatus > currentStatus {
		return newStatus
	}

	return currentStatus
}

type AccountResponse struct {
	Data  *Account `json:"data"`                                                          // Data for the account
	Error *string  `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this transaction
}

type AccountQueryFilter struct {
	Name     string `form:"name" filterField:"false"`   // Fuzzy filter for the account name
	Note     string `form:"note" filterField:"false"`   // Fuzzy filter for the note
	BudgetID string `form:"budget"`                     // By budget ID
	OnBudget bool   `form:"onBudget"`                   // Is the account on-budget?
	External bool   `form:"external"`                   // Is the account external?
	Archived bool   `form:"archived"`                   // Is the account archived?
	Search   string `form:"search" filterField:"false"` // By string in name or note
	Offset   uint   `form:"offset" filterField:"false"` // The offset of the first Account returned. Defaults to 0.
	Limit    int    `form:"limit" filterField:"false"`  // Maximum number of Accounts to return. Defaults to 50.
}

func (f AccountQueryFilter) model() (models.Account, error) {
	budgetID, err := httputil.UUIDFromString(f.BudgetID)
	if err != nil {
		return models.Account{}, err
	}

	return models.Account{
		BudgetID: budgetID,
		OnBudget: f.OnBudget,
		External: f.External,
		Archived: f.Archived,
	}, nil
}

type RecentEnvelopesResponse struct {
	Data  []RecentEnvelope `json:"data"`                                                          // Data for the account
	Error *string          `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this transaction
}

type RecentEnvelope struct {
	ID       *uuid.UUID `json:"id"`
	Name     string     `json:"name"`
	Archived bool       `json:"archived"`
}

type AccountComputedRequest struct {
	Time time.Time `form:"time"` // The time for which the computation is requested
	IDs  []string  `form:"ids"`  // A list of UUIDs for the accounts
}

type AccountComputedData struct {
	ID                uuid.UUID       `json:"id" example:"95018a69-758b-46c6-8bab-db70d9614f9d"` // ID of the account
	Balance           decimal.Decimal `json:"balance" example:"2735.17"`                         // Balance of the account, including all transactions referencing it
	ReconciledBalance decimal.Decimal `json:"reconciledBalance" example:"2539.57"`               // Balance of the account, including all reconciled transactions referencing it
}

type AccountComputedDataResponse struct {
	Data  []AccountComputedData `json:"data"`
	Error *string               `json:"error"`
}
