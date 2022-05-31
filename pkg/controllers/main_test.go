package controllers_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// // Environment for the test suite. Used to save the database connection
// type TestSuiteEnv struct {
// 	suite.Suite
// 	db *gorm.DB
// }

// // Pseudo-Test run by go test that runs the test suite
// func TestSuite(t *testing.T) {
// 	suite.Run(t, new(TestSuiteEnv))
// }

// // SetupSuite is called before any of the tests in the suite are run
// func (suite *TestSuiteEnv) SetupSuite() {
// 	database.Setup()
// 	suite.db = database.GetDB()
// }

// // SetupTest is called before each test in the suite
// func (suite *TestSuiteEnv) SetupTest() {
// }

// // TearDownTest is called after each test in the suite
// func (suite *TestSuiteEnv) TearDownTest() {
// 	database.ClearTable()
// }

// // TearDownSuite is called after all tests in the suite have run
// func (suite *TestSuiteEnv) TearDownSuite() {
// 	sqlDB, err := suite.db.DB()
// 	if err != nil {
// 		suite.Assert().FailNow("TearDownSuite: Database connection could not be closed", err)
// 	}

// 	sqlDB.Close()
// }

// TestMain takes care of the test setup for this package.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	// Always remove the DB after running tests
	defer os.Remove("data/gorm.db")

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode("release")
	}

	err := database.ConnectDatabase()
	if err != nil {
		log.Fatalf("Database connection failed with: %s", err.Error())
	}

	// Migrate all models so that the schema is correct
	err = database.DB.AutoMigrate(models.Budget{}, models.Account{}, models.Category{}, models.Envelope{}, models.Transaction{}, models.Allocation{})
	if err != nil {
		log.Fatalf("Database migration failed with: %s", err.Error())
	}

	budget := models.Budget{
		BudgetCreate: models.BudgetCreate{
			Name: "Testing Budget",
			Note: "GNU: Terry Pratchett",
		},
	}
	database.DB.Create(&budget)

	bankAccount := models.Account{
		AccountCreate: models.AccountCreate{
			Name:     "Bank Account",
			BudgetID: budget.ID,
			OnBudget: true,
		},
	}
	database.DB.Create(&bankAccount)

	cashAccount := models.Account{
		AccountCreate: models.AccountCreate{
			Name:     "Cash Account",
			BudgetID: budget.ID,
			OnBudget: false,
		},
	}
	database.DB.Create(&cashAccount)

	externalAccount := models.Account{
		AccountCreate: models.AccountCreate{
			Name:     "External Account",
			BudgetID: budget.ID,
			External: true,
		},
	}
	database.DB.Create(&externalAccount)

	category := models.Category{
		CategoryCreate: models.CategoryCreate{
			Name:     "Running costs",
			BudgetID: budget.ID,
			Note:     "For e.g. groceries and energy bills",
		},
	}
	database.DB.Create(&category)

	envelope := models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			Name:       "Utilities",
			CategoryID: category.ID,
			Note:       "Energy & Water",
		},
	}
	database.DB.Create(&envelope)

	allocationJan := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Month:      1,
			Year:       2022,
			Amount:     decimal.NewFromFloat(20.99),
		},
	}
	database.DB.Create(&allocationJan)

	allocationFeb := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Month:      2,
			Year:       2022,
			Amount:     decimal.NewFromFloat(47.12),
		},
	}
	database.DB.Create(&allocationFeb)

	allocationMar := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			EnvelopeID: envelope.ID,
			Month:      3,
			Year:       2022,
			Amount:     decimal.NewFromFloat(31.17),
		},
	}
	database.DB.Create(&allocationMar)

	waterBillTransactionJan := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
			Amount:               decimal.NewFromFloat(10.0),
			Note:                 "Water bill for January",
			BudgetID:             budget.ID,
			SourceAccountID:      bankAccount.ID,
			DestinationAccountID: externalAccount.ID,
			EnvelopeID:           envelope.ID,
			Reconciled:           true,
		},
	}
	database.DB.Create(&waterBillTransactionJan)

	waterBillTransactionFeb := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
			Amount:               decimal.NewFromFloat(5.0),
			Note:                 "Water bill for February",
			BudgetID:             budget.ID,
			SourceAccountID:      bankAccount.ID,
			DestinationAccountID: externalAccount.ID,
			EnvelopeID:           envelope.ID,
			Reconciled:           false,
		},
	}
	database.DB.Create(&waterBillTransactionFeb)

	waterBillTransactionMar := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
			Amount:               decimal.NewFromFloat(15.0),
			Note:                 "Water bill for March",
			BudgetID:             budget.ID,
			SourceAccountID:      bankAccount.ID,
			DestinationAccountID: externalAccount.ID,
			EnvelopeID:           envelope.ID,
			Reconciled:           false,
		},
	}

	database.DB.Create(&waterBillTransactionMar)

	return m.Run()
}
