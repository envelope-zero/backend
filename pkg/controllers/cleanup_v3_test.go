package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestCleanupV3() {
	_ = suite.createTestBudgetV3(suite.T(), models.BudgetCreate{})
	account := suite.createTestAccountV3(suite.T(), controllers.AccountCreateV3{Name: "TestCleanup"})
	_ = suite.createTestCategoryV3(suite.T(), controllers.CategoryCreateV3{})
	envelope := suite.createTestEnvelopeV3(suite.T(), controllers.EnvelopeCreateV3{})
	_ = suite.createTestTransactionV3(suite.T(), models.TransactionCreate{Amount: decimal.NewFromFloat(17.32)})
	_ = suite.patchTestMonthConfigV3(suite.T(), envelope.Data.ID, types.NewMonth(time.Now().Year(), time.Now().Month()), models.MonthConfigCreate{})
	_ = suite.createTestMatchRuleV3(suite.T(), models.MatchRuleCreate{AccountID: account.Data.ID, Match: "Delete me"})

	tests := []string{
		"http://example.com/v3/accounts",
		"http://example.com/v3/budgets",
		"http://example.com/v3/categories",
		"http://example.com/v3/envelopes",
		"http://example.com/v3/goals",
		"http://example.com/v3/match-rules",
		"http://example.com/v3/transactions",
	}

	// Delete
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v3?confirm=yes-please-delete-everything", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify
	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, tt, "")
			assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

			var response struct {
				Data []any `json:"data"`
			}

			suite.decodeResponse(&recorder, &response)
			assert.Len(t, response.Data, 0, "There are resources left for type %s", tt)
		})
	}
}

func (suite *TestSuiteStandard) TestCleanupV3Fails() {
	tests := []struct {
		name string
		path string
	}{
		{"Invalid path", "confirm=2"},
		{"Confirmation wrong", "confirm=invalid-confirmation"},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			recorder := test.Request(suite.controller, t, http.MethodDelete, fmt.Sprintf("http://example.com/v3?%s", tt.path), "")
			assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
		})
	}
}

func (suite *TestSuiteStandard) TestCleanupV3DBError() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v3?confirm=yes-please-delete-everything", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusInternalServerError)
}
