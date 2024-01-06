package models_test

import (
	"strings"

	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestMonthConfigSelf() {
	assert.Equal(suite.T(), "Month Config", models.MonthConfig{}.Self())
}

func (suite *TestSuiteStandard) TestMonthConfigTrimWhitespace() {
	note := " Some more whitespace in the notes    "

	account := suite.createTestMonthConfig(models.MonthConfig{
		Note: note,
		EnvelopeID: suite.createTestEnvelope(models.Envelope{
			CategoryID: suite.createTestCategory(models.Category{
				BudgetID: suite.createTestBudget(models.Budget{}).ID,
			}).ID,
		}).ID,
	})

	assert.Equal(suite.T(), strings.TrimSpace(note), account.Note)
}
