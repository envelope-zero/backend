package models_test

import (
	"testing"

	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
