package v4

import (
	"fmt"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionEditable struct {
	Date time.Time `json:"date" example:"1815-12-10T18:43:00.271152Z"` // Date of the transaction. Time is currently only used for sorting

	// The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
	Amount decimal.Decimal `json:"amount" example:"14.03" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The amount for the transaction

	Note                  string     `json:"note" example:"Lunch" default:""`                                     // A note
	BudgetID              uuid.UUID  `json:"budgetId" example:"55eecbd8-7c46-4b06-ada9-f287802fb05e"`             // ID of the budget
	SourceAccountID       uuid.UUID  `json:"sourceAccountId" example:"fd81dc45-a3a2-468e-a6fa-b2618f30aa45"`      // ID of the source account
	DestinationAccountID  uuid.UUID  `json:"destinationAccountId" example:"8e16b456-a719-48ce-9fec-e115cfa7cbcc"` // ID of the destination account
	EnvelopeID            *uuid.UUID `json:"envelopeId" example:"2649c965-7999-4873-ae16-89d5d5fa972e"`           // ID of the envelope
	ReconciledSource      bool       `json:"reconciledSource" example:"true" default:"false"`                     // Is the transaction reconciled in the source account?
	ReconciledDestination bool       `json:"reconciledDestination" example:"true" default:"false"`                // Is the transaction reconciled in the destination account?

	AvailableFrom types.Month `json:"availableFrom" example:"2021-11-17T00:00:00Z"` // The date from which on the transaction amount is available for budgeting. Only used for income transactions. Defaults to the transaction date.

	ImportHash string `json:"importHash" example:"867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70" default:""` // The SHA256 hash of a unique combination of values to use in duplicate detection
}

// model returns the database resource for the API representation of the editable fields
func (editable TransactionEditable) model() models.Transaction {
	return models.Transaction{
		Date:                  editable.Date,
		Amount:                editable.Amount,
		Note:                  editable.Note,
		BudgetID:              editable.BudgetID,
		SourceAccountID:       editable.SourceAccountID,
		DestinationAccountID:  editable.DestinationAccountID,
		EnvelopeID:            editable.EnvelopeID,
		ReconciledSource:      editable.ReconciledSource,
		ReconciledDestination: editable.ReconciledDestination,
		AvailableFrom:         editable.AvailableFrom,
		ImportHash:            editable.ImportHash,
	}
}

type TransactionLinks struct {
	Self string `json:"self" example:"https://example.com/api/v4/transactions/d430d7c3-d14c-4712-9336-ee56965a6673"` // The transaction itself
}

// Transaction is the representation of a Transaction in API v4.
type Transaction struct {
	models.DefaultModel
	TransactionEditable
	Links TransactionLinks `json:"links"`
}

// newTransaction returns the API v4 representation of the resource
func newTransaction(c *gin.Context, model models.Transaction) Transaction {
	url := c.GetString(string(models.DBContextURL))

	return Transaction{
		DefaultModel: model.DefaultModel,
		TransactionEditable: TransactionEditable{
			Date:                  model.Date,
			Amount:                model.Amount,
			Note:                  model.Note,
			BudgetID:              model.BudgetID,
			SourceAccountID:       model.SourceAccountID,
			DestinationAccountID:  model.DestinationAccountID,
			EnvelopeID:            model.EnvelopeID,
			ReconciledSource:      model.ReconciledSource,
			ReconciledDestination: model.ReconciledDestination,
			AvailableFrom:         model.AvailableFrom,
			ImportHash:            model.ImportHash,
		},
		Links: TransactionLinks{
			Self: fmt.Sprintf("%s/v4/transactions/%s", url, model.ID),
		},
	}
}

type TransactionListResponse struct {
	Data       []Transaction `json:"data"`                                                          // List of transactions
	Error      *string       `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination   `json:"pagination"`                                                    // Pagination information
}

type TransactionCreateResponse struct {
	Error *string               `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []TransactionResponse `json:"data"`                                                          // List of created Transactions
}

func (t *TransactionCreateResponse) appendError(err httperrors.Error, status int) int {
	s := err.Error()
	t.Data = append(t.Data, TransactionResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	if err.Status > status {
		status = err.Status
	}

	return status
}

type TransactionResponse struct {
	Error *string      `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this transaction
	Data  *Transaction `json:"data"`                                                          // The Transaction data, if creation was successful
}

type TransactionQueryFilter struct {
	Date                  time.Time       `form:"date" filterField:"false"`              // Exact date. Time is ignored.
	FromDate              time.Time       `form:"fromDate" filterField:"false"`          // From this date. Time is ignored.
	UntilDate             time.Time       `form:"untilDate" filterField:"false"`         // Until this date. Time is ignored.
	Amount                decimal.Decimal `form:"amount"`                                // Exact amount
	AmountLessOrEqual     decimal.Decimal `form:"amountLessOrEqual" filterField:"false"` // Amount less than or equal to this
	AmountMoreOrEqual     decimal.Decimal `form:"amountMoreOrEqual" filterField:"false"` // Amount more than or equal to this
	Note                  string          `form:"note" filterField:"false"`              // Note contains this string
	BudgetID              string          `form:"budget"`                                // ID of the budget
	SourceAccountID       string          `form:"source"`                                // ID of the source account
	DestinationAccountID  string          `form:"destination"`                           // ID of the destination account
	EnvelopeID            string          `form:"envelope"`                              // ID of the envelope
	ReconciledSource      bool            `form:"reconciledSource"`                      // Is the transaction reconciled in the source account?
	ReconciledDestination bool            `form:"reconciledDestination"`                 // Is the transaction reconciled in the destination account?
	AccountID             string          `form:"account" filterField:"false"`           // ID of either source or destination account
	Offset                uint            `form:"offset" filterField:"false"`            // The offset of the first Transaction returned. Defaults to 0.
	Limit                 int             `form:"limit" filterField:"false"`             // Maximum number of transactions to return. Defaults to 50.
}

func (f TransactionQueryFilter) model() (models.Transaction, httperrors.Error) {
	budgetID, err := httputil.UUIDFromString(f.BudgetID)
	if !err.Nil() {
		return models.Transaction{}, err
	}

	sourceAccountID, err := httputil.UUIDFromString(f.SourceAccountID)
	if !err.Nil() {
		return models.Transaction{}, err
	}

	destinationAccountID, err := httputil.UUIDFromString(f.DestinationAccountID)
	if !err.Nil() {
		return models.Transaction{}, err
	}

	envelopeID, err := httputil.UUIDFromString(f.EnvelopeID)
	if !err.Nil() {
		return models.Transaction{}, err
	}

	// If the envelopeID is nil, use an actual nil, not uuid.Nil
	var eID *uuid.UUID
	if envelopeID != uuid.Nil {
		eID = &envelopeID
	}

	// This does not set the string or date fields since they are
	// handled in the controller function
	return TransactionEditable{
		Amount:                f.Amount,
		BudgetID:              budgetID,
		SourceAccountID:       sourceAccountID,
		DestinationAccountID:  destinationAccountID,
		EnvelopeID:            eID,
		ReconciledSource:      f.ReconciledSource,
		ReconciledDestination: f.ReconciledDestination,
	}.model(), httperrors.Error{}
}
