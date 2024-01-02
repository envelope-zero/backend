package models_test

import (
	"strconv"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestAccountTrimWhitespace() {
	name := "\t Whitespace galore!   "
	note := " Some more whitespace in the notes    "
	importHash := "  867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70  \t"

	account := suite.createTestAccount(models.AccountCreate{
		Name:       name,
		Note:       note,
		ImportHash: importHash,
		BudgetID:   suite.createTestBudget(models.BudgetCreate{}).ID,
	})

	assert.Equal(suite.T(), strings.TrimSpace(name), account.Name)
	assert.Equal(suite.T(), strings.TrimSpace(note), account.Note)
	assert.Equal(suite.T(), strings.TrimSpace(importHash), account.ImportHash)
}

func (suite *TestSuiteStandard) TestAccountCalculations() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	initialBalanceDate := time.Now()

	account := suite.createTestAccount(models.AccountCreate{
		Name:               "TestAccountCalculations",
		BudgetID:           budget.ID,
		OnBudget:           true,
		External:           false,
		InitialBalance:     decimal.NewFromFloat(170),
		InitialBalanceDate: &initialBalanceDate,
	})

	externalAccount := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		External: true,
	})

	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		CategoryID: category.ID,
	})

	incomingTransaction := suite.createTestTransaction(models.Transaction{
		BudgetID:              budget.ID,
		EnvelopeID:            &envelope.ID,
		SourceAccountID:       externalAccount.ID,
		DestinationAccountID:  account.ID,
		ReconciledDestination: true,
		Amount:                decimal.NewFromFloat(32.17),
	})

	outgoingTransaction := suite.createTestTransaction(models.Transaction{
		BudgetID:             budget.ID,
		EnvelopeID:           &envelope.ID,
		SourceAccountID:      account.ID,
		DestinationAccountID: externalAccount.ID,
		Amount:               decimal.NewFromFloat(17.45),
	})

	_ = suite.createTestTransaction(models.Transaction{
		BudgetID:             budget.ID,
		SourceAccountID:      externalAccount.ID,
		DestinationAccountID: account.ID,
		Amount:               decimal.NewFromFloat(100),
		Date:                 time.Now(),
		AvailableFrom:        types.MonthOf(time.Now()).AddDate(0, 1),
		Note:                 "Future Income Transaction",
	})

	balance, _, err := account.GetBalanceMonth(suite.db, types.Month{})
	assert.Nil(suite.T(), err)

	reconciled, err := account.SumReconciled(suite.db)
	assert.Nil(suite.T(), err)

	expected := incomingTransaction.Amount.Sub(outgoingTransaction.Amount).Add(account.InitialBalance).Add(decimal.NewFromFloat(100)) // Add 100 for futureIncomeTransaction
	assert.True(suite.T(), balance.Equal(expected), "Balance for account is not correct. Should be: %v but is %v", expected, balance)

	expected = incomingTransaction.Amount.Add(account.InitialBalance)
	assert.True(suite.T(), reconciled.Equal(expected), "Reconciled balance for account is not correct. Should be: %v but is %v", expected, reconciled)

	balanceNow, availableNow, err := account.GetBalanceMonth(suite.db, types.MonthOf(time.Now()))
	assert.Nil(suite.T(), err)

	expected = decimal.NewFromFloat(284.72)
	assert.True(suite.T(), balanceNow.Equal(expected), "Current balance for account is not correct. Should be: %v but is %v", expected, balanceNow)

	expected = decimal.NewFromFloat(184.72)
	assert.True(suite.T(), availableNow.Equal(expected), "Available balance for account is not correct. Should be: %v but is %v", expected, availableNow)

	err = suite.db.Delete(&incomingTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be deleted", err)
	}

	balance, _, err = account.GetBalanceMonth(suite.db, types.Month{})
	assert.Nil(suite.T(), err)

	reconciled, err = account.SumReconciled(suite.db)
	assert.Nil(suite.T(), err)

	expected = outgoingTransaction.Amount.Neg().Add(account.InitialBalance).Add(decimal.NewFromFloat(100)) // Add 100 for futureIncomeTransaction
	assert.True(suite.T(), balance.Equal(expected), "Balance for account is not correct. Should be: %v but is %v", expected, balance)

	expected = decimal.NewFromFloat(0).Add(account.InitialBalance)
	assert.True(suite.T(), reconciled.Equal(expected), "Reconciled balance for account is not correct. Should be: %v but is %v", expected, reconciled)
}

func (suite *TestSuiteStandard) TestAccountTransactions() {
	account := models.Account{}

	transactions := account.Transactions(suite.db)
	assert.Len(suite.T(), transactions, 0)
}

func (suite *TestSuiteStandard) TestAccountOnBudget() {
	account := models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: true,
		},
	}

	err := account.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.False(suite.T(), account.OnBudget, "OnBudget is true even though the account is external")

	account = models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: false,
		},
	}

	err = account.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.True(suite.T(), account.OnBudget, "OnBudget is set to false even though the account is internal")
}

func (suite *TestSuiteStandard) TestAccountGetBalanceMonthDBFail() {
	account := models.Account{}

	suite.CloseDB()

	_, _, err := account.GetBalanceMonth(suite.db, types.NewMonth(2017, 7))
	suite.Assert().NotNil(err)
	suite.Assert().Equal("sql: database is closed", err.Error())
}

// TestAccountDuplicateNames ensures that two accounts cannot have the same name.
func (suite *TestSuiteStandard) TestAccountDuplicateNames() {
	budget := suite.createTestBudget(models.BudgetCreate{})

	_ = suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		Name:     "TestAccountDuplicateNames",
	})

	externalAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			Name:     "TestAccountDuplicateNames",
			External: true,
		},
	}
	err := suite.db.Save(&externalAccount).Error
	if err == nil {
		suite.Assert().Fail("Account with the same name than another account could be saved. This must not be possible", err)
		return
	}

	suite.Assert().Contains(err.Error(), "UNIQUE constraint failed: accounts.name, accounts.budget_id", "Error message for account creation fail does not match expected message")
}

func (suite *TestSuiteStandard) TestAccountOnBudgetToOnBudgetTransactionsNoEnvelopes() {
	budget := suite.createTestBudget(models.BudgetCreate{
		Name: "TestAccountOnBudgetToOnBudgetTransactionsNoEnvelopes",
	})

	account := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		OnBudget: true,
		External: false,
		Name:     "TestAccountOnBudgetToOnBudgetTransactionsNoEnvelopes",
	})

	transferTargetAccount := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		OnBudget: false,
		External: false,
		Name:     "TestAccountOnBudgetToOnBudgetTransactionsNoEnvelopes:Target",
	})

	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		CategoryID: category.ID,
	})

	t := suite.createTestTransaction(models.Transaction{
		Amount:               decimal.NewFromFloat(17.23),
		BudgetID:             budget.ID,
		SourceAccountID:      account.ID,
		DestinationAccountID: transferTargetAccount.ID,
		EnvelopeID:           &envelope.ID,
	})

	// Try saving the account, which must fail
	data := models.Account{AccountCreate: models.AccountCreate{OnBudget: true}}
	err := suite.db.Model(&transferTargetAccount).Select("OnBudget").Updates(data).Error

	if !assert.NotNil(suite.T(), err, "Target account could be updated to be on budget while having transactions with envelopes being set") {
		assert.FailNow(suite.T(), "Exiting because assertion was not met")
	}
	assert.Contains(suite.T(), err.Error(), "the account cannot be set to on budget because the following transactions have an envelope set: ")

	// Update the envelope for the transaction
	t.EnvelopeID = nil
	err = suite.db.Model(&t).Updates(&t).Error
	assert.Nil(suite.T(), err, "Transaction could not be updated")

	// Save again
	err = suite.db.Model(&transferTargetAccount).Updates(&transferTargetAccount).Error
	assert.Nil(suite.T(), err, "Target account could not be updated despite transaction having its envelope removed")
}

func (suite *TestSuiteStandard) TestAccountOffBudgetToOnBudgetTransactionsNoEnvelopes() {
	budget := suite.createTestBudget(models.BudgetCreate{
		Name: "TestAccountOffBudgetToOnBudgetTransactionsNoEnvelopes",
	})

	account := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		OnBudget: false,
		External: false,
		Name:     "TestAccountOffBudgetToOnBudgetTransactionsNoEnvelopes",
	})

	transferTargetAccount := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		OnBudget: false,
		External: false,
		Name:     "TestAccountOffBudgetToOnBudgetTransactionsNoEnvelopes:Target",
	})

	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		CategoryID: category.ID,
	})

	_ = suite.createTestTransaction(models.Transaction{
		Amount:               decimal.NewFromFloat(17.23),
		BudgetID:             budget.ID,
		SourceAccountID:      account.ID,
		DestinationAccountID: transferTargetAccount.ID,
		EnvelopeID:           &envelope.ID,
	})

	// Try saving the account, which must work
	data := models.Account{AccountCreate: models.AccountCreate{OnBudget: true}}
	err := suite.db.Model(&transferTargetAccount).Select("OnBudget").Updates(data).Error

	assert.Nil(suite.T(), err, "Target account could not be updated to be on budget, but it does not have transactions with envelopes being set")
}

func (suite *TestSuiteStandard) TestAccountSelf() {
	assert.Equal(suite.T(), "Account", models.Account{}.Self())
}

func (suite *TestSuiteStandard) TestAccountRecentEnvelopes() {
	budget := suite.createTestBudget(models.BudgetCreate{})

	account := suite.createTestAccount(models.AccountCreate{
		BudgetID:       budget.ID,
		Name:           "Internal Account",
		OnBudget:       true,
		External:       false,
		InitialBalance: decimal.NewFromFloat(170),
	})

	externalAccount := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		Name:     "External Account",
		External: true,
	})

	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
	})

	envelopeIDs := []*uuid.UUID{}
	for i := 0; i < 3; i++ {
		envelope := suite.createTestEnvelope(models.EnvelopeCreate{
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
			BudgetID:             budget.ID,
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
			BudgetID:             budget.ID,
			EnvelopeID:           nil,
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: account.ID,
			Amount:               decimal.NewFromFloat(1337.42),
		})
	}

	ids, err := account.RecentEnvelopes(suite.db)
	if err != nil {
		suite.Assert().FailNow("Could not compute recent envelopes", err)
	}

	if !suite.Assert().Len(ids, 4, "The number of envelopes in recentEnvelopes is not correct, expected 4, got %d", len(ids)) {
		suite.FailNow("Incorrect envelope number")
	}

	// The last envelope needs to be the first in the sort since it
	// has been the most common one
	suite.Assert().Equal(envelopeIDs[2], ids[0])

	// Income is the second one since it appears three times
	var nilUUIDPointer *uuid.UUID
	suite.Assert().Equal(nilUUIDPointer, ids[1])

	// Order for envelopes with the same frequency is undefined
}
