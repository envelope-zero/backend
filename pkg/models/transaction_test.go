package models_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *TestSuiteStandard) TestTransactionCreate() {
	budgetID := suite.createTestBudget(models.Budget{}).ID
	envelopeID := uuid.New()

	tests := []struct {
		name                 string
		amount               float64
		sourceAccountID      uuid.UUID
		destinationAccountID uuid.UUID
		envelopeID           *uuid.UUID
		err                  error
	}{
		{
			"Valid",
			17,
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			nil,
			nil,
		},
		{
			"Invalid source",
			17,
			uuid.New(),
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			nil,
			models.ErrTransactionInvalidSourceAccount,
		},
		{
			"Invalid destination",
			17,
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			uuid.New(),
			nil,
			models.ErrTransactionInvalidDestinationAccount,
		},
		{
			"Invalid amount",
			0,
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			nil,
			models.ErrTransactionAmountNotPositive,
		},
		{
			"No internal accounts",
			100,
			suite.createTestAccount(models.Account{BudgetID: budgetID, External: true}).ID,
			suite.createTestAccount(models.Account{BudgetID: budgetID, External: true}).ID,
			nil,
			models.ErrTransactionNoInternalAccounts,
		},
		{
			"Transfer with Envelope Set",
			100,
			suite.createTestAccount(models.Account{BudgetID: budgetID, OnBudget: true}).ID,
			suite.createTestAccount(models.Account{BudgetID: budgetID, OnBudget: true}).ID,
			&envelopeID,
			models.ErrTransactionTransferBetweenOnBudgetWithEnvelope,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			transaction := models.Transaction{
				Amount:               decimal.NewFromFloat(tt.amount),
				SourceAccountID:      tt.sourceAccountID,
				DestinationAccountID: tt.destinationAccountID,
				EnvelopeID:           tt.envelopeID,
			}
			err := models.DB.Create(&transaction).Error
			assert.ErrorIs(t, err, tt.err, "Error is: %s", err)
		})
	}
}

func (suite *TestSuiteStandard) TestTransactionUpdate() {
	budgetID := suite.createTestBudget(models.Budget{}).ID
	transaction := suite.createTestTransaction(models.Transaction{
		Amount:               decimal.NewFromFloat(17),
		SourceAccountID:      suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
		DestinationAccountID: suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
	})

	tests := []struct {
		name                 string
		amount               float64
		sourceAccountID      uuid.UUID
		destinationAccountID uuid.UUID
		err                  error
	}{
		{
			"Invalid source",
			17,
			uuid.New(),
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			models.ErrTransactionInvalidSourceAccount,
		},
		{
			"Invalid destination",
			17,
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			uuid.New(),
			models.ErrTransactionInvalidDestinationAccount,
		},
		{
			"Invalid amount",
			0,
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
			models.ErrTransactionAmountNotPositive,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			update := models.Transaction{
				Amount:               decimal.NewFromFloat(tt.amount),
				SourceAccountID:      tt.sourceAccountID,
				DestinationAccountID: tt.destinationAccountID,
			}
			err := models.DB.Model(&transaction).Updates(update).Error
			assert.ErrorIs(t, err, tt.err, "Error is: %s", err)
		})
	}
}

func (suite *TestSuiteStandard) TestTransactionTrimWhitespace() {
	note := " Some more whitespace in the notes    "
	importHash := "  867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70  \t"

	budgetID := suite.createTestBudget(models.Budget{}).ID

	transaction := suite.createTestTransaction(models.Transaction{
		Amount:               decimal.NewFromFloat(17),
		Note:                 note,
		ImportHash:           importHash,
		SourceAccountID:      suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
		DestinationAccountID: suite.createTestAccount(models.Account{BudgetID: budgetID}).ID,
	})

	assert.Equal(suite.T(), strings.TrimSpace(note), transaction.Note)
	assert.Equal(suite.T(), strings.TrimSpace(importHash), transaction.ImportHash)
}

func (suite *TestSuiteStandard) TestTransactionSaveTime() {
	budget := suite.createTestBudget(models.Budget{})
	internalAccount := suite.createTestAccount(models.Account{External: false, BudgetID: budget.ID})
	externalAccount := suite.createTestAccount(models.Account{External: true, BudgetID: budget.ID})

	now := time.Now()

	transaction := models.Transaction{
		SourceAccountID:      internalAccount.ID,
		DestinationAccountID: externalAccount.ID,
		Date:                 now.In(time.UTC),
	}
	err := transaction.BeforeSave(models.DB)
	if err != nil {
		assert.Fail(suite.T(), "transaction.BeforeSave failed", err)
	}

	assert.True(suite.T(), transaction.Date.Equal(now), "Transaction Date not correct!")
}

// TestTransactionReconciled verifies that the Transaction.BeforeSave method
// correctly enforces ReconciledSource and ReconciledDestination to be false
// when the respective account is external.
func (suite *TestSuiteStandard) TestTransactionReconciled() {
	budget := suite.createTestBudget(models.Budget{})
	internalAccount := suite.createTestAccount(models.Account{External: false, BudgetID: budget.ID})
	externalAccount := suite.createTestAccount(models.Account{External: true, BudgetID: budget.ID})

	tests := []struct {
		name                      string
		sourceAccountID           uuid.UUID
		source                    models.Account
		setReconciledSource       bool
		wantReconciledSource      bool
		destinationAccountID      uuid.UUID
		destination               models.Account
		setReconciledDestination  bool
		wantReconciledDestination bool
		expectedError             string
	}{
		{"ReconciledDestination enforced false for external", internalAccount.ID, models.Account{}, true, true, externalAccount.ID, models.Account{}, true, false, ""},
		{"ReconciledSource enforced false for external", externalAccount.ID, models.Account{}, true, false, internalAccount.ID, models.Account{}, true, true, ""},
		{"ReconciledDestination enforced false for external, SourceAccount & DestinationAccount set", internalAccount.ID, internalAccount, true, true, externalAccount.ID, externalAccount, true, false, ""},
		{"ReconciledSource enforced false for external, SourceAccount & DestinationAccount set", externalAccount.ID, externalAccount, true, false, internalAccount.ID, internalAccount, true, true, ""},
		{"SourceAccount does not exist", uuid.New(), models.Account{}, true, false, internalAccount.ID, models.Account{}, false, false, "no existing account with specified SourceAccountID: there is no account matching your query"},
		{"DestinationAccount does not exist", externalAccount.ID, externalAccount, false, false, uuid.New(), models.Account{}, true, false, "no existing account with specified DestinationAccountID: there is no account matching your query"},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			transaction := models.Transaction{
				SourceAccount:         tt.source,
				DestinationAccount:    tt.destination,
				Note:                  tt.name,
				SourceAccountID:       tt.sourceAccountID,
				DestinationAccountID:  tt.destinationAccountID,
				ReconciledSource:      tt.setReconciledSource,
				ReconciledDestination: tt.setReconciledDestination,
			}

			err := transaction.BeforeSave(models.DB)
			if err != nil {
				if tt.expectedError == "" {
					assert.Fail(t, "transaction.BeforeSave failed", err)
				} else {
					if !strings.Contains(err.Error(), tt.expectedError) {
						assert.Failf(t, "Wrong error in transaction.BeforeSave", "transaction.BeforeSave returned a wrong error of '%s', expected it to contain '%s'", err.Error(), tt.expectedError)
					}
				}

				// Error was either handled correctly or the test has already failed
				return
			}

			assert.Equal(t, tt.wantReconciledSource, transaction.ReconciledSource, "ReconciledSource is wrong")
			assert.Equal(t, tt.wantReconciledDestination, transaction.ReconciledDestination, "ReconciledSource is wrong")
		})
	}
}

// Regression test for https://github.com/envelope-zero/backend/issues/768
func (suite *TestSuiteStandard) TestTransactionAvailableFromDate() {
	budget := suite.createTestBudget(models.Budget{})
	internalAccount := suite.createTestAccount(models.Account{External: false, BudgetID: budget.ID})
	externalAccount := suite.createTestAccount(models.Account{External: true, BudgetID: budget.ID})

	transaction := models.Transaction{
		SourceAccountID:      externalAccount.ID,
		DestinationAccountID: internalAccount.ID,
		Note:                 "TestTransactionAvailableFromDate",
		AvailableFrom:        types.NewMonth(2023, 9),
		Date:                 time.Date(2023, 10, 7, 0, 0, 0, 0, time.UTC),
	}

	err := models.DB.Save(&transaction).Error
	suite.Assert().NotNil(err, "Saving a transaction with an AvailableFrom date in a month before the transaction date should not be possible")
	suite.Assert().Contains(err.Error(), "availability month must not be earlier than the month of the transaction")
}

// TestTransactionEnvelopeNilUUID is a regression test to ensure that when the API receives a
// nil UUID "00000000-0000-0000-0000-000000000000" for the envelope,
// it is set to nil for the transaction resource.
//
// If it were not, it would reference the envelope with the nil UUID, which does not exist.
func (suite *TestSuiteStandard) TestTransactionEnvelopeNilUUID() {
	budget := suite.createTestBudget(models.Budget{})
	internalAccount := suite.createTestAccount(models.Account{External: false, BudgetID: budget.ID})
	externalAccount := suite.createTestAccount(models.Account{External: true, BudgetID: budget.ID})

	eID := uuid.Nil

	transaction := models.Transaction{
		Amount:               decimal.NewFromFloat(42),
		SourceAccountID:      externalAccount.ID,
		DestinationAccountID: internalAccount.ID,
		EnvelopeID:           &eID,
		Note:                 "TestTransactionEnvelopeNilUUID",
	}

	err := models.DB.Save(&transaction).Error
	suite.Assert().Nil(err, "Saving a transaction with a nil UUID for the Envelope ID should not result in an error")
}

func (suite *TestSuiteStandard) TestTransactionExport() {
	t := suite.T()

	budget := suite.createTestBudget(models.Budget{})
	internalAccount := suite.createTestAccount(models.Account{External: false, BudgetID: budget.ID})
	externalAccount := suite.createTestAccount(models.Account{External: true, BudgetID: budget.ID})

	for range 2 {
		_ = suite.createTestTransaction(models.Transaction{SourceAccountID: internalAccount.ID, DestinationAccountID: externalAccount.ID, Amount: decimal.NewFromFloat(10)})
	}

	raw, err := models.Transaction{}.Export()
	if err != nil {
		require.Fail(t, "transaction export failed", err)
	}

	var transactions []models.Transaction
	err = json.Unmarshal(raw, &transactions)
	if err != nil {
		require.Fail(t, "JSON could not be unmarshaled", err)
	}

	require.Len(t, transactions, 2, "number of transactions in export is wrong")
}
