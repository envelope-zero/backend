package ynab4_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"testing"
	"testing/iotest"
	"time"

	"github.com/envelope-zero/backend/v5/internal/importer"
	"github.com/envelope-zero/backend/v5/internal/importer/parser/ynab4"
	"github.com/envelope-zero/backend/v5/internal/models"
	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

// date returns a time.Time for a specific date at midnight UTC.
func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// testDB returns an in-memory test database and a function to close it.
func testDB(t *testing.T) (*gorm.DB, func() error) {
	// Connect a database
	err := models.Connect(test.TmpFile(t))
	if err != nil {
		log.Fatalf("Database connection failed with: %#v", err)
	}

	// Create the context and store the API URL
	ctx := context.Background()
	url, _ := url.Parse("https://example.com")
	ctx = context.WithValue(ctx, models.DBContextURL, url)

	sqlDB, _ := models.DB.DB()
	return models.DB.WithContext(ctx), sqlDB.Close
}

func TestParseNoFile(t *testing.T) {
	_, err := ynab4.Parse(iotest.ErrReader(errors.New("Some reading error")))
	assert.NotNil(t, err, "Expected file opening to fail")
	assert.Contains(t, err.Error(), "could not read data from file", "Wrong error on parsing broken file: %s", err)
}

func TestParseFail(t *testing.T) {
	tests := []struct {
		name string // The file name. Used as test name, too
		err  string // The expected error message
	}{
		{"CorruptNonParseableHidden", "hidden category could not be parsed"},
		{"EmptyFile", "not a valid YNAB4 Budget.yfull file"},
		{"CorruptNonParseableTransactionDate", "error parsing transactions: could not parse date"},
		{"CorruptMonthlyBudget", "parsing time \"2022-12-01-12\" as \"2006-01-02T15:04:05Z07:00\""},
		{"CorruptNoMatchingTransfer", "could not find corresponding transaction"},
		{"CorruptMissingTargetTransaction", "could not find corresponding transaction for sub-transaction transfer"},
	}

	for _, tt := range tests {
		f, err := os.OpenFile(fmt.Sprintf("../../../../test/data/importer/%s.yfull", tt.name), os.O_RDONLY, 0o400)
		if err != nil {
			assert.FailNow(t, "Failed to open the test file", err)
		}

		_, err = ynab4.Parse(f)
		assert.NotNil(t, err, "Expected parsing to fail")
		assert.Contains(t, err.Error(), tt.err, "Wrong error on parsing broken file: %s", err)
	}
}

// TestParse parses a full budget and then verifies that all resources exist.
//
// Screenshots for the Budget.yfull file opened in YNAB 4 are in the test/data/importer directory
// for easier verification of future features and bugs.
func TestParse(t *testing.T) {
	f, err := os.OpenFile("../../../../test/data/importer/Budget.yfull", os.O_RDONLY, 0o400)
	require.Nil(t, err, "Failed to open the test file: %w", err)

	// Call the parser
	r, err := ynab4.Parse(f)
	require.Nil(t, err, "Parsing failed", err)

	// Create test database and import
	db, closeDb := testDB(t)
	defer closeDb()

	b, err := importer.Create(db, r)

	// Check correctness of import
	require.Nil(t, err)
	assert.Equal(t, "€", b.Currency, "Currency is wrong")

	// Check accounts
	var accounts []models.Account
	db.Find(&accounts)
	t.Run("accounts", func(t *testing.T) {
		testAccounts(t, accounts)
	})

	// Check MatchRules
	var matchRules []models.MatchRule
	db.Find(&matchRules)
	t.Run("MatchRules", func(t *testing.T) {
		testMatchRules(t, matchRules, accounts)
	})

	// Check categories
	var categories []models.Category
	db.Find(&categories)
	t.Run("categories", func(t *testing.T) {
		testCategories(t, categories)
	})

	// Check envelopes
	var envelopes []models.Envelope
	db.Find(&envelopes)
	t.Run("envelopes", func(t *testing.T) {
		testEnvelopes(t, categories, envelopes)
	})

	// Check transactions
	var transactions []models.Transaction
	db.Find(&transactions)
	t.Run("transactions", func(t *testing.T) {
		testTransactions(t, accounts, envelopes, transactions)
	})
}

// testAccount tests all account resources.
func testAccounts(t *testing.T, accounts []models.Account) {
	// - 5 internal accounts
	// - 14 external accounts imported from YNAB payees
	// - 1 external account "YNAB 4 Import - No Payee" for transactions without payee
	assert.Len(t, accounts, 22, "Number of accounts is wrong")

	// Check number of internal accounts. This implicitly checks the number of external
	// accounts, too as we already check the total number above.
	var count int
	for _, a := range accounts {
		if !a.External {
			count++
		}
	}
	assert.Equal(t, 6, count, "Count of internal and external accounts does not match")

	// Check account details
	tests := []struct {
		name               string
		initialBalance     float32
		initialBalanceDate time.Time
		onBudget           bool
		archived           bool
		note               string
	}{
		{"Checking", 100, time.Date(2022, 10, 15, 0, 0, 0, 0, time.UTC), true, false, ""},
		{"Cash", 21.17, time.Date(2022, 10, 16, 0, 0, 0, 0, time.UTC), true, false, "Money I carry in my pocket"},
		{"Second Checking", -200, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true, false, ""},
		{"Savings", 0, time.Time{}, false, false, ""},
		{"Accidental Account", 0, time.Time{}, true, true, "This person has an account they accidentally opened.\n\nIt has a few bucks in it."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := slices.IndexFunc(accounts, func(a models.Account) bool { return a.Name == tt.name })
			require.NotEqual(t, -1, idx, "No account with expected name")

			a := accounts[idx]
			assert.True(t, a.InitialBalance.Equal(decimal.NewFromFloat32(tt.initialBalance)), "Initial balance does not match, is %s, expected %f", a.InitialBalance, tt.initialBalance)
			assert.False(t, a.External, "Account is marked external")
			assert.Equal(t, tt.onBudget, a.OnBudget, "On Budget is wrong")
			assert.Equal(t, tt.archived, a.Archived, "Archived is wrong")
			assert.Equal(t, tt.note, a.Note, "Note differs. Should be '%s', but is '%s'", tt.note, a.Note)

			if tt.initialBalance != 0 {
				assert.Equal(t, &tt.initialBalanceDate, a.InitialBalanceDate, "Initial balance date does not match")
			}
		})
	}
}

// testMatchRules tests all MatchRule resources.
func testMatchRules(t *testing.T, matchRules []models.MatchRule, accounts []models.Account) {
	assert.Len(t, matchRules, 5, "Number of MatchRules is wrong")

	// Check MatchRule details
	//
	// Not checking priority because YNAB4 does not have priorities here
	// Therefore we always set it to 0 on import
	tests := []struct {
		match   string
		account string
	}{
		{"Mum*", "Parents"},
		{"*& Dad", "Parents"},
		{"My Parents", "Parents"},
		{"Co", "Favorite Coffee Shop"},
		{"*Coffee Shop*", "Favorite Coffee Shop"},
	}

	for _, tt := range tests {
		t.Run(tt.match, func(t *testing.T) {
			// Find Account
			aIdx := slices.IndexFunc(accounts, func(a models.Account) bool { return a.Name == tt.account })
			require.NotEqual(t, -1, aIdx, "No Account with the name the Match Rule is targeting")

			// Find Match Rule
			mIdx := slices.IndexFunc(matchRules, func(m models.MatchRule) bool { return m.Match == tt.match })
			require.NotEqual(t, -1, mIdx, "No Match Rule with the match we are looking for")

			a := accounts[aIdx]
			m := matchRules[mIdx]

			assert.Equal(t, a.ID, m.AccountID, "Match Rule Account ID and actual Account ID do not match")
		})
	}
}

// testCategories tests all the categories for correct import.
func testCategories(t *testing.T, categories []models.Category) {
	// 3 categories, 1 (Rainy Day Funds) only has archived envelopes
	assert.Len(t, categories, 3, "Number of categories is wrong")

	tests := []struct {
		name     string
		note     string
		archived bool
	}{
		{"Savings Goals", "Money I'm saving for big expenses", false},
		{"Everyday Expenses", "", false},
		{"Rainy Day Funds", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := slices.IndexFunc(categories, func(c models.Category) bool { return c.Name == tt.name })
			require.NotEqual(t, -1, idx, "No category with expected name")

			assert.Equal(t, tt.archived, categories[idx].Archived)
			assert.Equal(t, tt.note, categories[idx].Note)
			assert.Equal(t, tt.name, categories[idx].Name)
		})
	}
}

// testEnvelopes tests all envelope resources for correctness.
func testEnvelopes(t *testing.T, categories []models.Category, envelopes []models.Envelope) {
	assert.Len(t, envelopes, 11, "Number of envelopes is wrong")

	tests := []struct {
		name     string
		category string
		note     string
		archived bool
	}{
		{"Groceries", "Everyday Expenses", "", false},
		{"Transport", "Everyday Expenses", "", false},
		{"Spending Money", "Everyday Expenses", "", false},
		{"Restaurants", "Everyday Expenses", "This includes food to go, ice cream parlors etc.", false},
		{"Medical", "Everyday Expenses", "", false},
		{"Clothing", "Everyday Expenses", "", false},
		{"Household Goods", "Everyday Expenses", "", false},
		{"Banking", "Everyday Expenses", "", false},
		{"Car Replacement", "Savings Goals", "", false},
		{"Vacation", "Savings Goals", "", false},
		{"Health Insurance", "Rainy Day Funds", "", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s: %s", tt.category, tt.name), func(t *testing.T) {
			idx := slices.IndexFunc(categories, func(c models.Category) bool { return c.Name == tt.category })
			require.NotEqual(t, -1, idx, "No category with expected name for this envelope")

			idx = slices.IndexFunc(envelopes, func(e models.Envelope) bool { return e.Name == tt.name && e.CategoryID == categories[idx].ID })
			require.NotEqual(t, -1, idx, "No envelope with expected name and category")
			e := envelopes[idx]

			assert.Equal(t, tt.note, e.Note, "Note differs, is '%s', should be '%s'", e.Note, tt.note)
			assert.Equal(t, tt.archived, e.Archived, "Archived is wrong")
		})
	}
}

// testTransactions tests the imported transactions.
//
// It assumes that there is only one transaction per day with the same note.
func testTransactions(t *testing.T, accounts []models.Account, envelopes []models.Envelope, transactions []models.Transaction) {
	// 27 transactions total in YNAB 4 (counting each sub-transaction as 1)
	// subtract 5 Starting balance transactions
	// subtract 5 transfers (since transfers in EZ are only one transaction, not 2)
	assert.Len(t, transactions, 17, "Number of transactions is wrong")

	tests := []struct {
		date                       time.Time
		amount                     float32
		note                       string
		sourceAccount              string
		sourceAccountExternal      bool
		destinationAccount         string
		destinationAccountExternal bool
		envelope                   string
		reconciledSource           bool
		reconciledDestination      bool
		availableFrom              types.Month
	}{
		{date(2022, 10, 10), 120, "", "Checking", false, "Hospital", true, "Medical", false, false, types.Month{}},
		{date(2022, 10, 20), 15, "", "Checking", false, "Checking (External)", true, "Restaurants", true, false, types.Month{}},
		{date(2022, 10, 21), 50, "", "Checking", false, "Savings", false, "Vacation", true, true, types.Month{}},
		{date(2022, 10, 21), 10, "Put in too much", "Savings", false, "Checking", false, "Vacation", true, false, types.Month{}},
		{date(2022, 10, 25), 1000, "", "Employer", true, "Checking", false, "", false, true, types.NewMonth(2022, 11)},
		{date(2022, 11, 1), 30, "Sweatpants", "Checking", false, "Online Shop", true, "Clothing", true, false, types.Month{}},
		{date(2022, 11, 1), 120, "Kitchen Appliance", "Checking", false, "Online Shop", true, "Household Goods", true, false, types.Month{}},
		{date(2022, 11, 10), 100, "Needed some cash", "Checking", false, "Cash", false, "", false, true, types.Month{}},
		{date(2022, 11, 10), 5, "Needed some cash: Withdrawal Fee", "Checking", false, "YNAB 4 Import - No Payee", true, "Spending Money", false, false, types.Month{}},
		{date(2022, 11, 11), 20, "Taking some back out", "Savings", false, "Checking", false, "Vacation", true, false, types.Month{}},
		{date(2022, 11, 11), 50, "Grandma gave me 50 bucks for a new mixer", "YNAB 4 Import - No Payee", true, "Checking", false, "Household Goods", false, false, types.Month{}},
		{date(2022, 11, 15), 95, "Compensation for returned goods", "Online Platform", true, "Checking", false, "", false, false, types.NewMonth(2022, 12)},
		{date(2022, 11, 15), 15, "", "Checking", false, "Online Platform", true, "Clothing", false, false, types.Month{}},
		{date(2022, 11, 28), 10, "", "Checking", false, "Accidental Account", false, "", false, false, types.Month{}},
		{date(2022, 12, 15), 10, "", "Cash", false, "Takeout", true, "Restaurants", false, false, types.Month{}},
		{date(2022, 12, 30), 20, "", "Checking", false, "Cash", false, "", false, false, types.Month{}},
		{date(2022, 12, 31), 100, "Car is slowly breaking down", "Checking", false, "Savings", false, "", false, false, types.Month{}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s: %s -> %s - %s", tt.date, tt.sourceAccount, tt.destinationAccount, tt.note), func(t *testing.T) {
			// Get transaction
			idx := slices.IndexFunc(transactions, func(t models.Transaction) bool { return t.Date == tt.date && t.Note == tt.note })
			require.NotEqual(t, -1, idx, "No transaction at expected date with expected note")
			tr := transactions[idx]

			// Get source account
			idx = slices.IndexFunc(accounts, func(a models.Account) bool {
				return a.Name == tt.sourceAccount && a.External == tt.sourceAccountExternal
			})
			require.NotEqual(t, -1, idx, "Source account not found in account list")
			source := accounts[idx]

			// Get destination account
			idx = slices.IndexFunc(accounts, func(a models.Account) bool {
				return a.Name == tt.destinationAccount && a.External == tt.destinationAccountExternal
			})
			require.NotEqual(t, -1, idx, "Destination account not found in account list")
			destination := accounts[idx]

			// Get envelope, only if set
			if tt.envelope != "" {
				idx = slices.IndexFunc(envelopes, func(e models.Envelope) bool { return e.Name == tt.envelope })
				require.NotEqual(t, -1, idx, "Envelope not found in envelope list")
				envelope := envelopes[idx]
				assert.Equal(t, &envelope.ID, tr.EnvelopeID, "Envelope ID is not correct, is %s, should be %s", tr.EnvelopeID, &envelope.ID)
			}

			assert.Equal(t, source.ID, tr.SourceAccountID, "Source account ID is not correct, is %s, should be %s", tr.SourceAccountID, source.ID)
			assert.Equal(t, destination.ID, tr.DestinationAccountID, "Destination account ID is not correct, is %s, should be %s", tr.DestinationAccountID, destination.ID)
			assert.True(t, decimal.NewFromFloat32(tt.amount).Equal(tr.Amount), "Amount does not match. Is %s, expected %f", tr.Amount, tt.amount)
			assert.Equal(t, tt.note, tr.Note, "Note differs. Should be '%s', but is '%s'", tt.note, tr.Note)
			assert.Equal(t, tt.reconciledSource, tr.ReconciledSource, "ReconciledSource flag is wrong")
			assert.Equal(t, tt.reconciledDestination, tr.ReconciledDestination, "ReconciledDestination flag is wrong")

			// Only check availableFrom if it is set
			if !tt.availableFrom.Equal(types.Month{}) {
				assert.Equal(t, tt.availableFrom, tr.AvailableFrom, "Available from does not match. Is %s, expected %s", tr.AvailableFrom, tt.availableFrom)
			}
		})
	}
}
