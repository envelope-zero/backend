package models_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v5/internal/models"
	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *TestSuiteStandard) TestAccountTrimWhitespace() {
	name := "\t Whitespace galore!   "
	note := " Some more whitespace in the notes    "
	importHash := "  867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70  \t"

	account := suite.createTestAccount(models.Account{
		Name:       name,
		Note:       note,
		ImportHash: importHash,
		BudgetID:   suite.createTestBudget(models.Budget{}).ID,
	})

	assert.Equal(suite.T(), strings.TrimSpace(name), account.Name)
	assert.Equal(suite.T(), strings.TrimSpace(note), account.Note)
	assert.Equal(suite.T(), strings.TrimSpace(importHash), account.ImportHash)
}

func (suite *TestSuiteStandard) TestAccountCalculations() {
	budget := suite.createTestBudget(models.Budget{})
	initialBalanceDate := time.Now()

	account := suite.createTestAccount(models.Account{
		Name:               "TestAccountCalculations",
		BudgetID:           budget.ID,
		OnBudget:           true,
		External:           false,
		InitialBalance:     decimal.NewFromFloat(170),
		InitialBalanceDate: &initialBalanceDate,
	})

	externalAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		External: true,
	})

	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		CategoryID: category.ID,
	})

	incomingTransaction := suite.createTestTransaction(models.Transaction{
		EnvelopeID:            &envelope.ID,
		SourceAccountID:       externalAccount.ID,
		DestinationAccountID:  account.ID,
		ReconciledDestination: true,
		Amount:                decimal.NewFromFloat(32.17),
	})

	outgoingTransaction := suite.createTestTransaction(models.Transaction{
		EnvelopeID:           &envelope.ID,
		SourceAccountID:      account.ID,
		DestinationAccountID: externalAccount.ID,
		Amount:               decimal.NewFromFloat(17.45),
	})

	_ = suite.createTestTransaction(models.Transaction{
		SourceAccountID:      externalAccount.ID,
		DestinationAccountID: account.ID,
		Amount:               decimal.NewFromFloat(100),
		Date:                 time.Now(),
		AvailableFrom:        types.MonthOf(time.Now()).AddDate(0, 1),
		Note:                 "Future Income Transaction",
	})

	balance, _, err := account.GetBalanceMonth(models.DB, types.Month{})
	assert.Nil(suite.T(), err)

	reconciled, err := account.ReconciledBalance(models.DB, time.Now().AddDate(1, 0, 0))
	assert.Nil(suite.T(), err)

	expected := incomingTransaction.Amount.Sub(outgoingTransaction.Amount).Add(account.InitialBalance).Add(decimal.NewFromFloat(100)) // Add 100 for futureIncomeTransaction
	assert.True(suite.T(), balance.Equal(expected), "Balance for account is not correct. Should be: %v but is %v", expected, balance)

	expected = incomingTransaction.Amount.Add(account.InitialBalance)
	assert.True(suite.T(), reconciled.Equal(expected), "Reconciled balance for account is not correct. Should be: %v but is %v", expected, reconciled)

	balanceNow, availableNow, err := account.GetBalanceMonth(models.DB, types.MonthOf(time.Now()))
	assert.Nil(suite.T(), err)

	expected = decimal.NewFromFloat(284.72)
	assert.True(suite.T(), balanceNow.Equal(expected), "Current balance for account is not correct. Should be: %v but is %v", expected, balanceNow)

	expected = decimal.NewFromFloat(184.72)
	assert.True(suite.T(), availableNow.Equal(expected), "Available balance for account is not correct. Should be: %v but is %v", expected, availableNow)

	err = models.DB.Delete(&incomingTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be deleted", err)
	}

	balance, _, err = account.GetBalanceMonth(models.DB, types.Month{})
	assert.Nil(suite.T(), err)

	reconciled, err = account.ReconciledBalance(models.DB, time.Now().AddDate(1, 0, 0))
	assert.Nil(suite.T(), err)

	balanceOnly, err := account.Balance(models.DB, time.Now().AddDate(1, 0, 0)) // Adding a year so that we cover all transactions
	assert.Nil(suite.T(), err)

	reconciledOnly, err := account.ReconciledBalance(models.DB, time.Now())
	assert.Nil(suite.T(), err)

	expected = outgoingTransaction.Amount.Neg().Add(account.InitialBalance).Add(decimal.NewFromFloat(100)) // Add 100 for futureIncomeTransaction
	assert.True(suite.T(), balance.Equal(expected), "Balance for account is not correct. Should be: %v but is %v", expected, balance)
	assert.True(suite.T(), balanceOnly.Equal(expected), "Balance for account is not correct. Should be: %v but is %v", expected, balanceOnly)

	expected = decimal.NewFromFloat(0).Add(account.InitialBalance)
	assert.True(suite.T(), reconciled.Equal(expected), "Reconciled balance for account is not correct. Should be: %v but is %v", expected, reconciled)
	assert.True(suite.T(), reconciledOnly.Equal(expected), "Reconciled balance for account is not correct. Should be: %v but is %v", expected, reconciledOnly)
}

func (suite *TestSuiteStandard) TestAccountTransactions() {
	account := models.Account{}

	transactions := account.Transactions(models.DB)
	assert.Len(suite.T(), transactions, 0)
}

func (suite *TestSuiteStandard) TestAccountOnBudget() {
	account := models.Account{
		OnBudget: true,
		External: true,
	}

	err := account.BeforeSave(models.DB)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.False(suite.T(), account.OnBudget, "OnBudget is true even though the account is external")

	account = models.Account{
		OnBudget: true,
		External: false,
	}

	err = account.BeforeSave(models.DB)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.True(suite.T(), account.OnBudget, "OnBudget is set to false even though the account is internal")
}

func (suite *TestSuiteStandard) TestAccountGetBalanceMonthDBFail() {
	account := models.Account{}

	suite.CloseDB()

	_, _, err := account.GetBalanceMonth(models.DB, types.NewMonth(2017, 7))
	suite.Assert().ErrorIs(err, models.ErrGeneral)
}

// TestAccountDuplicateNames ensures that two accounts cannot have the same name.
func (suite *TestSuiteStandard) TestAccountDuplicateNames() {
	budget := suite.createTestBudget(models.Budget{})

	_ = suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		Name:     "TestAccountDuplicateNames",
	})

	externalAccount := models.Account{
		BudgetID: budget.ID,
		Name:     "TestAccountDuplicateNames",
		External: true,
	}
	err := models.DB.Save(&externalAccount).Error
	if err == nil {
		suite.Assert().Fail("Account with the same name than another account could be saved. This must not be possible", err)
		return
	}

	suite.Assert().ErrorIs(err, models.ErrAccountNameNotUnique)
}

func (suite *TestSuiteStandard) TestAccountOnBudgetToOnBudgetTransactionsNoEnvelopes() {
	budget := suite.createTestBudget(models.Budget{
		Name: "TestAccountOnBudgetToOnBudgetTransactionsNoEnvelopes",
	})

	account := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		OnBudget: true,
		External: false,
		Name:     "TestAccountOnBudgetToOnBudgetTransactionsNoEnvelopes",
	})

	transferTargetAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		OnBudget: false,
		External: false,
		Name:     "TestAccountOnBudgetToOnBudgetTransactionsNoEnvelopes:Target",
	})

	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		CategoryID: category.ID,
	})

	t := suite.createTestTransaction(models.Transaction{
		Amount:               decimal.NewFromFloat(17.23),
		SourceAccountID:      account.ID,
		DestinationAccountID: transferTargetAccount.ID,
		EnvelopeID:           &envelope.ID,
	})

	// Try saving the account, which must fail
	data := models.Account{OnBudget: true}
	err := models.DB.Model(&transferTargetAccount).Select("OnBudget").Updates(data).Error

	require.NotNil(suite.T(), err, "Target account could be updated to be on budget while having transactions with envelopes being set")
	assert.Contains(suite.T(), err.Error(), "the account cannot be set to on budget because the following transactions have an envelope set: ")

	// Update the envelope for the transaction
	t.EnvelopeID = nil
	err = models.DB.Model(&t).Updates(t).Error
	assert.Nil(suite.T(), err, "Transaction could not be updated")

	// Save again
	err = models.DB.Model(&transferTargetAccount).Updates(transferTargetAccount).Error
	assert.Nil(suite.T(), err, "Target account could not be updated despite transaction having its envelope removed")
}

func (suite *TestSuiteStandard) TestAccountOffBudgetToOnBudgetTransactionsNoEnvelopes() {
	budget := suite.createTestBudget(models.Budget{
		Name: "TestAccountOffBudgetToOnBudgetTransactionsNoEnvelopes",
	})

	account := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		OnBudget: false,
		External: false,
		Name:     "TestAccountOffBudgetToOnBudgetTransactionsNoEnvelopes",
	})

	transferTargetAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		OnBudget: false,
		External: false,
		Name:     "TestAccountOffBudgetToOnBudgetTransactionsNoEnvelopes:Target",
	})

	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		CategoryID: category.ID,
	})

	_ = suite.createTestTransaction(models.Transaction{
		Amount:               decimal.NewFromFloat(17.23),
		SourceAccountID:      account.ID,
		DestinationAccountID: transferTargetAccount.ID,
		EnvelopeID:           &envelope.ID,
	})

	// Try saving the account, which must work
	data := models.Account{OnBudget: true}
	err := models.DB.Model(&transferTargetAccount).Select("OnBudget").Updates(data).Error

	assert.Nil(suite.T(), err, "Target account could not be updated to be on budget, but it does not have transactions with envelopes being set")
}

func (suite *TestSuiteStandard) TestAccountRecentEnvelopes() {
	budget := suite.createTestBudget(models.Budget{})

	account := suite.createTestAccount(models.Account{
		BudgetID:       budget.ID,
		Name:           "Internal Account",
		OnBudget:       true,
		External:       false,
		InitialBalance: decimal.NewFromFloat(170),
	})

	externalAccount := suite.createTestAccount(models.Account{
		BudgetID: budget.ID,
		Name:     "External Account",
		External: true,
	})

	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	envelopeIDs := []*uuid.UUID{}
	for i := 0; i < 3; i++ {
		envelope := suite.createTestEnvelope(models.Envelope{
			CategoryID: category.ID,
			Name:       strconv.Itoa(i),
		})

		envelopeIDs = append(envelopeIDs, &envelope.ID)

		// Sleep for 10 milliseconds because we only save timestamps with 1 millisecond accuracy
		// This is needed because the test runs so fast that all envelopes are sometimes created
		// within the same millisecond, making the result non-deterministic
		time.Sleep(1 * time.Millisecond)
	}

	// Create 15 transactions:
	//  * 2 for the first envelope
	//  * 2 for the second envelope
	//  * 11 for the last envelope
	for i := 0; i < 15; i++ {
		eIndex := i
		if i > 5 {
			eIndex = 2
		}
		_ = suite.createTestTransaction(models.Transaction{
			EnvelopeID:           envelopeIDs[eIndex%3],
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: account.ID,
			Amount:               decimal.NewFromFloat(17.45),
		})
	}

	// Create three income transactions
	//
	// This is a regression test for income always showing at the last
	// position in the recent envelopes (before the LIMIT) since count(id) for
	// income was always 0. This is due to the envelope ID for income being NULL
	// and count() not counting NULL values.
	//
	// Creating three income transactions puts "income" as the second most common
	// envelope, verifying the fix
	for i := 0; i < 3; i++ {
		_ = suite.createTestTransaction(models.Transaction{
			EnvelopeID:           nil,
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: account.ID,
			Amount:               decimal.NewFromFloat(1337.42),
		})
	}

	ids, err := account.RecentEnvelopes(models.DB)
	if err != nil {
		suite.Assert().FailNow("Could not compute recent envelopes", err)
	}

	suite.Require().Len(ids, 4, "The number of envelopes in recentEnvelopes is not correct, expected 4, got %d", len(ids), "Incorrect envelope number")

	// The last envelope needs to be the first in the sort since it
	// has been the most common one
	suite.Assert().Equal(envelopeIDs[2], ids[0])

	// Income is the second one since it appears three times
	var nilUUIDPointer *uuid.UUID
	suite.Assert().Equal(nilUUIDPointer, ids[1])

	// Order for envelopes with the same frequency is undefined
}

func (suite *TestSuiteStandard) TestAccountExport() {
	t := suite.T()

	budget := suite.createTestBudget(models.Budget{
		Name: "TestAccountExport",
	})

	for range 2 {
		_ = suite.createTestAccount(models.Account{BudgetID: budget.ID})
	}

	raw, err := models.Account{}.Export()
	if err != nil {
		require.Fail(t, "account export failed", err)
	}

	var accounts []models.Account
	err = json.Unmarshal(raw, &accounts)
	if err != nil {
		require.Fail(t, "JSON could not be unmarshaled", err)
	}

	require.Len(t, accounts, 2, "Number of accounts in export is wrong")
}
