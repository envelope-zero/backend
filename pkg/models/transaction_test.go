package models_test

import (
	"strings"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestTransactionFindTimeUTC() {
	tz, _ := time.LoadLocation("Europe/Berlin")

	transaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
		},
	}

	err := transaction.AfterFind(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "transaction.AfterFind failed", err)
	}

	assert.Equal(suite.T(), time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")
}

func (suite *TestSuiteStandard) TestTransactionSaveTimeUTC() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	internalAccount := suite.createTestAccount(models.AccountCreate{External: false, BudgetID: budget.ID})
	externalAccount := suite.createTestAccount(models.AccountCreate{External: true, BudgetID: budget.ID})

	tz, _ := time.LoadLocation("Europe/Berlin")

	transaction := models.Transaction{TransactionCreate: models.TransactionCreate{SourceAccountID: internalAccount.ID, DestinationAccountID: externalAccount.ID}}
	err := transaction.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "transaction.BeforeSave failed", err)
	}

	assert.Equal(suite.T(), time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")

	transaction = models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
		},
	}
	err = transaction.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "transaction.BeforeSave failed", err)
	}

	assert.Equal(suite.T(), time.UTC, transaction.Date.Location(), "Timezone for model is not UTC")
}

// TestTransactionReconciled verifies that the Transaction.BeforeSave method
// correctly enforces ReconciledSource and ReconciledDestination to be false
// when the respective account is external.
func (suite *TestSuiteStandard) TestTransactionReconciled() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	internalAccount := suite.createTestAccount(models.AccountCreate{External: false, BudgetID: budget.ID})
	externalAccount := suite.createTestAccount(models.AccountCreate{External: true, BudgetID: budget.ID})

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
		{"SourceAccount does not exist", uuid.New(), models.Account{}, true, false, internalAccount.ID, models.Account{}, false, false, "no existing account with specified SourceAccountID: record not found"},
		{"DestinationAccount does not exist", externalAccount.ID, externalAccount, false, false, uuid.New(), models.Account{}, true, false, "no existing account with specified DestinationAccountID: record not found"},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			transaction := models.Transaction{
				SourceAccount:      tt.source,
				DestinationAccount: tt.destination,
				TransactionCreate: models.TransactionCreate{
					Note:                  tt.name,
					SourceAccountID:       tt.sourceAccountID,
					DestinationAccountID:  tt.destinationAccountID,
					ReconciledSource:      tt.setReconciledSource,
					ReconciledDestination: tt.setReconciledDestination,
				},
			}

			err := transaction.BeforeSave(suite.db)
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

func (suite *TestSuiteStandard) TestTransactionSelf() {
	assert.Equal(suite.T(), "Transaction", models.Transaction{}.Self())
}

// Regression test for https://github.com/envelope-zero/backend/issues/768
func (suite *TestSuiteStandard) TestTransactionAvailableFromDate() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	internalAccount := suite.createTestAccount(models.AccountCreate{External: false, BudgetID: budget.ID})
	externalAccount := suite.createTestAccount(models.AccountCreate{External: true, BudgetID: budget.ID})

	transaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: internalAccount.ID,
			Note:                 "TestTransactionAvailableFromDate",
			AvailableFrom:        types.NewMonth(2023, 9),
			Date:                 time.Date(2023, 10, 7, 0, 0, 0, 0, time.UTC),
		},
	}

	err := suite.db.Save(&transaction).Error
	suite.Assert().NotNil(err, "Saving a transaction with an AvailableFrom date in a month before the transaction date should not be possible")
	suite.Assert().Contains(err.Error(), "availability month must not be earlier than the month of the transaction")
}
