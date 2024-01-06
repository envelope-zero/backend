package v4_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	v4 "github.com/envelope-zero/backend/v4/pkg/controllers/v4"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestCleanup() {
	_ = suite.createTestBudget(suite.T(), v4.BudgetEditable{})
	account := suite.createTestAccount(suite.T(), models.Account{Name: "TestCleanup"})
	_ = suite.createTestCategory(suite.T(), v4.CategoryEditable{})
	envelope := suite.createTestEnvelope(suite.T(), v4.EnvelopeEditable{})
	_ = suite.createTestTransaction(suite.T(), models.Transaction{Amount: decimal.NewFromFloat(17.32)})
	_ = suite.patchTestMonthConfig(suite.T(), envelope.Data.ID, types.NewMonth(time.Now().Year(), time.Now().Month()), v4.MonthConfigEditable{})
	_ = suite.createTestMatchRule(suite.T(), models.MatchRule{AccountID: account.Data.ID, Match: "Delete me"})

	tests := []string{
		"http://example.com/v4/accounts",
		"http://example.com/v4/budgets",
		"http://example.com/v4/categories",
		"http://example.com/v4/envelopes",
		"http://example.com/v4/goals",
		"http://example.com/v4/match-rules",
		"http://example.com/v4/transactions",
	}

	// Delete
	recorder := test.Request(suite.T(), http.MethodDelete, "http://example.com/v4?confirm=yes-please-delete-everything", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify
	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(suite.T(), http.MethodGet, tt, "")
			test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)

			var response struct {
				Data []any `json:"data"`
			}

			test.DecodeResponse(t, &recorder, &response)
			assert.Len(t, response.Data, 0, "There are resources left for type %s", tt)
		})
	}
}

func (suite *TestSuiteStandard) TestCleanupFails() {
	tests := []struct {
		name string
		path string
	}{
		{"Invalid path", "confirm=2"},
		{"Confirmation wrong", "confirm=invalid-confirmation"},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			recorder := test.Request(t, http.MethodDelete, fmt.Sprintf("http://example.com/v4?%s", tt.path), "")
			test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
		})
	}
}

func (suite *TestSuiteStandard) TestCleanupDBError() {
	suite.CloseDB()

	recorder := test.Request(suite.T(), http.MethodDelete, "http://example.com/v4?confirm=yes-please-delete-everything", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusInternalServerError)
}
