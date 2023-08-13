package models_test

import (
	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestMonthConfigHooks() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.ID, Name: "Upkeep"})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.ID, Name: "Utilities"})
	mConfig := suite.createTestMonthConfig(models.MonthConfig{EnvelopeID: envelope.ID, Month: types.NewMonth(2022, 10)})

	err := mConfig.AfterFind(suite.db)
	assert.Nil(suite.T(), err)

	err = mConfig.AfterSave(suite.db)
	assert.Nil(suite.T(), err)
}
