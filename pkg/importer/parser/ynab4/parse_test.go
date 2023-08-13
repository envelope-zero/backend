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

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/importer"
	"github.com/envelope-zero/backend/v3/pkg/importer/parser/ynab4"
	"github.com/envelope-zero/backend/v3/pkg/models"
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
func testDB() (*gorm.DB, func() error) {
	// Connect a database
	db, err := database.Connect(":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Database connection failed with: %#v", err)
	}

	models.Migrate(db)
	if err != nil {
		log.Fatalf("Database migration failed with: %#v", err)
	}

	// Create the context and store the API URL
	ctx := context.Background()
	url, _ := url.Parse("https://example.com")
	ctx = context.WithValue(ctx, database.ContextURL, url)

	sqlDB, _ := db.DB()
	return db.WithContext(ctx), sqlDB.Close
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
		{"CorruptMonthlyBudget", "error parsing budget allocations: could not parse date"},
		{"CorruptNoMatchingTransfer", "could not find corresponding transaction"},
		{"CorruptMissingTargetTransaction", "could not find corresponding transaction for sub-transaction transfer"},
	}

	for _, tt := range tests {
		f, err := os.OpenFile(fmt.Sprintf("../../../../testdata/importer/%s.yfull", tt.name), os.O_RDONLY, 0o400)
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
// Screenshots for the Budget.yfull file opened in YNAB 4 are in the testdata/importer directory
// for easier verification of future features and bugs.
func TestParse(t *testing.T) {
	f, err := os.OpenFile("../../../../testdata/importer/Budget.yfull", os.O_RDONLY, 0o400)
	require.Nil(t, err, "Failed to open the test file: %w", err)

	// Call the parser
	r, err := ynab4.Parse(f)
	require.Nil(t, err, "Parsing failed", err)

	// Create test database and import
	db, closeDb := testDB()
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

	// In YNAB 4, starting balance counts as income our outflow, in Envelope Zero it does not
	// Therefore, the numbers for available, balance, spent and income will differ in some cases
	tests := []struct {
		month     types.Month
		available float32
		balance   float32
		spent     float32
		budgeted  float32
		income    float32
	}{
		{types.NewMonth(2022, 10), 46.17, -100, -175, 75, 0},
		{types.NewMonth(2022, 11), 906.17, -60, -100, 140, 1000},
		{types.NewMonth(2022, 12), 886.17, -55, -110, 115, 95},
		{types.NewMonth(2023, 1), 576.17, 55, 0, 0, 0},
		{types.NewMonth(2023, 2), 456.17, 175, 0, 0, 0},
	}

	for _, tt := range tests {
		m, err := b.Month(db, tt.month)
		assert.Nil(t, err)

		assert.True(t, decimal.NewFromFloat32(tt.available).Equal(m.Available), "Available for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.available), m.Available)
		assert.True(t, decimal.NewFromFloat32(tt.balance).Equal(m.Balance), "Balance for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.balance), m.Balance)
		assert.True(t, decimal.NewFromFloat32(tt.spent).Equal(m.Spent), "Spent for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.spent), m.Spent)
		assert.True(t, decimal.NewFromFloat32(tt.budgeted).Equal(m.Budgeted), "Budgeted for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.budgeted), m.Budgeted)
		assert.True(t, decimal.NewFromFloat32(tt.income).Equal(m.Income), "Income for %s is wrong, should be %s but is %s", tt.month, decimal.NewFromFloat32(tt.income), m.Income)
	}
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
		hidden             bool
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
			if !assert.NotEqual(t, -1, idx, "No account with expected name") {
				return
			}

			a := accounts[idx]
			assert.True(t, a.InitialBalance.Equal(decimal.NewFromFloat32(tt.initialBalance)), "Initial balance does not match, is %s, expected %f", a.InitialBalance, tt.initialBalance)
			assert.False(t, a.External, "Account is marked external")
			assert.Equal(t, tt.onBudget, a.OnBudget, "On Budget is wrong")
			assert.Equal(t, tt.hidden, a.Hidden, "Hidden is wrong")
			assert.Equal(t, tt.note, a.Note, "Note differs. Should be '%s', but is '%s'", tt.note, a.Note)

			if tt.initialBalance != 0 {
				assert.Equal(t, &tt.initialBalanceDate, a.InitialBalanceDate, "Initial balance date does not match")
			}
		})
	}
}

// testCategories tests all the categories for correct import.
func testCategories(t *testing.T, categories []models.Category) {
	// 3 categories, 1 (Rainy Day Funds) only has hidden envelopes
	assert.Len(t, categories, 3, "Number of categories is wrong")

	tests := []struct {
		name   string
		note   string
		hidden bool
	}{
		{"Savings Goals", "Money I'm saving for big expenses", false},
		{"Everyday Expenses", "", false},
		{"Rainy Day Funds", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := slices.IndexFunc(categories, func(c models.Category) bool { return c.Name == tt.name })
			if !assert.NotEqual(t, -1, idx, "No category with expected name") {
				return
			}
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
		hidden   bool
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
			if !assert.NotEqual(t, -1, idx, "No category with expected name for this envelope") {
				return
			}

			idx = slices.IndexFunc(envelopes, func(e models.Envelope) bool { return e.Name == tt.name && e.CategoryID == categories[idx].ID })
			if !assert.NotEqual(t, -1, idx, "No envelope with expected name and category") {
				return
			}
			e := envelopes[idx]

			assert.Equal(t, tt.note, e.Note, "Note differs, is '%s', should be '%s'", e.Note, tt.note)
			assert.Equal(t, tt.hidden, e.Hidden, "Hidden is wrong")
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
			if !assert.NotEqual(t, -1, idx, "No transaction at expected date with expected note") {
				return
			}
			tr := transactions[idx]

			// Get source account
			idx = slices.IndexFunc(accounts, func(a models.Account) bool {
				return a.Name == tt.sourceAccount && a.External == tt.sourceAccountExternal
			})
			if !assert.NotEqual(t, -1, idx, "Source account not found in account list") {
				return
			}
			source := accounts[idx]

			// Get destination account
			idx = slices.IndexFunc(accounts, func(a models.Account) bool {
				return a.Name == tt.destinationAccount && a.External == tt.destinationAccountExternal
			})
			if !assert.NotEqual(t, -1, idx, "Destination account not found in account list") {
				return
			}
			destination := accounts[idx]

			// Get envelope, only if set
			if tt.envelope != "" {
				idx = slices.IndexFunc(envelopes, func(e models.Envelope) bool { return e.Name == tt.envelope })
				if !assert.NotEqual(t, -1, idx, "Envelope not found in envelope list") {
					return
				}
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
