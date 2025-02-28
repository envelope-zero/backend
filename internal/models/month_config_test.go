package models_test

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/envelope-zero/backend/v7/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func (suite *TestSuiteStandard) TestMonthConfigExport() {
	t := suite.T()

	budget := suite.createTestBudget(models.Budget{})
	category := suite.createTestCategory(models.Category{BudgetID: budget.ID})
	envelope := suite.createTestEnvelope(models.Envelope{CategoryID: category.ID})

	_ = suite.createTestMonthConfig(models.MonthConfig{EnvelopeID: envelope.ID, Month: types.NewMonth(1977, time.January)})
	_ = suite.createTestMonthConfig(models.MonthConfig{EnvelopeID: envelope.ID, Month: types.NewMonth(1977, time.February)})

	raw, err := models.MonthConfig{}.Export()
	if err != nil {
		require.Fail(t, "month config export failed", err)
	}

	var monthConfigs []models.MonthConfig
	err = json.Unmarshal(raw, &monthConfigs)
	if err != nil {
		require.Fail(t, "JSON could not be unmarshaled", err)
	}

	require.Len(t, monthConfigs, 2, "Number of monthc configs in export is wrong")
}
