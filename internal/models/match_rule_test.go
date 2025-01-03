package models_test

import (
	"encoding/json"
	"testing"

	"github.com/envelope-zero/backend/v5/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *TestSuiteStandard) TestMatchRuleBeforeCreate() {
	_ = suite.createTestMatchRule(models.MatchRule{
		AccountID: suite.createTestAccount(models.Account{
			BudgetID: suite.createTestBudget(models.Budget{}).ID,
		}).ID,
	})
}

func (suite *TestSuiteStandard) TestMatchRuleBeforeUpdate() {
	matchRule := suite.createTestMatchRule(models.MatchRule{
		AccountID: suite.createTestAccount(models.Account{
			BudgetID: suite.createTestBudget(models.Budget{}).ID,
		}).ID,
	})

	tests := []struct {
		name      string
		accountID uuid.UUID
		err       error
	}{
		{
			"Update account",
			suite.createTestAccount(models.Account{
				BudgetID: suite.createTestBudget(models.Budget{}).ID,
			}).ID,
			nil,
		},
		{
			"Update account to non-existing",
			uuid.New(),
			models.ErrResourceNotFound,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			err := models.DB.Model(&matchRule).Select("AccountID").Updates(models.MatchRule{AccountID: tt.accountID}).Error
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func (suite *TestSuiteStandard) TestMatchRuleExport() {
	t := suite.T()

	budget := suite.createTestBudget(models.Budget{})
	account := suite.createTestAccount(models.Account{BudgetID: budget.ID})

	for range 2 {
		_ = suite.createTestMatchRule(models.MatchRule{AccountID: account.ID})
	}

	raw, err := models.MatchRule{}.Export()
	if err != nil {
		require.Fail(t, "match rule export failed", err)
	}

	var matchRules []models.MatchRule
	err = json.Unmarshal(raw, &matchRules)
	if err != nil {
		require.Fail(t, "JSON could not be unmarshaled", err)
	}

	require.Len(t, matchRules, 2, "number of match rules in export is wrong")
}
