package controllers_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// TestMain takes care of the test setup for this package.
func TestMain(m *testing.M) {
	// Always remove the DB after running tests
	defer os.Remove("data/gorm.db")

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode("release")
	}

	err := models.ConnectDatabase()
	if err != nil {
		log.Fatalf("Database migration failed with: %s", err.Error())
	}

	budget := models.Budget{
		BudgetCreate: models.BudgetCreate{
			Name: "Testing Budget",
			Note: "GNU: Terry Pratchett",
		},
	}
	models.DB.Create(&budget)

	bankAccount := models.Account{
		Name:     "Bank Account",
		BudgetID: budget.ID,
		OnBudget: true,
		Balance:  decimal.NewFromFloat(13.37), // To verify this is not saved in the database
	}
	models.DB.Create(&bankAccount)

	cashAccount := models.Account{
		Name:     "Cash Account",
		BudgetID: budget.ID,
		OnBudget: false,
	}
	models.DB.Create(&cashAccount)

	externalAccount := models.Account{
		Name:     "External Account",
		BudgetID: budget.ID,
		External: true,
	}
	models.DB.Create(&externalAccount)

	category := models.Category{
		Name:     "Running costs",
		BudgetID: budget.ID,
		Note:     "For e.g. groceries and energy bills",
	}
	models.DB.Create(&category)

	envelope := models.Envelope{
		Name:       "Utilities",
		CategoryID: category.ID,
		Note:       "Energy & Water",
	}
	models.DB.Create(&envelope)

	allocationJan := models.Allocation{
		EnvelopeID: envelope.ID,
		Month:      1,
		Year:       2022,
		Amount:     decimal.NewFromFloat(20.99),
	}
	models.DB.Create(&allocationJan)

	allocationFeb := models.Allocation{
		EnvelopeID: envelope.ID,
		Month:      2,
		Year:       2022,
		Amount:     decimal.NewFromFloat(47.12),
	}
	models.DB.Create(&allocationFeb)

	allocationMar := models.Allocation{
		EnvelopeID: envelope.ID,
		Month:      3,
		Year:       2022,
		Amount:     decimal.NewFromFloat(31.17),
	}
	models.DB.Create(&allocationMar)

	waterBillTransactionJan := models.Transaction{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.ID,
		SourceAccountID:      bankAccount.ID,
		DestinationAccountID: externalAccount.ID,
		EnvelopeID:           envelope.ID,
		Reconciled:           true,
	}
	models.DB.Create(&waterBillTransactionJan)

	waterBillTransactionFeb := models.Transaction{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.ID,
		SourceAccountID:      bankAccount.ID,
		DestinationAccountID: externalAccount.ID,
		EnvelopeID:           envelope.ID,
		Reconciled:           false,
	}
	models.DB.Create(&waterBillTransactionFeb)

	waterBillTransactionMar := models.Transaction{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.ID,
		SourceAccountID:      bankAccount.ID,
		DestinationAccountID: externalAccount.ID,
		EnvelopeID:           envelope.ID,
		Reconciled:           false,
	}
	models.DB.Create(&waterBillTransactionMar)

	m.Run()
}
