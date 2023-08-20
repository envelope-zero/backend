package models_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestCategoryArchiveArchivesEnvelopes() {
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: suite.createTestBudget(models.BudgetCreate{}).ID,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		CategoryID: category.ID,
		Hidden:     false,
	})
	assert.False(suite.T(), envelope.Hidden, "Envelope archived on creation, it should not be")

	// Archive the category
	err := suite.db.Model(&category).Select("Hidden").Updates(models.Category{CategoryCreate: models.CategoryCreate{Hidden: true}}).Error
	assert.Nil(suite.T(), err)

	// Verify that the envelope is not archived
	err = suite.db.First(&envelope, envelope.ID).Error
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelope.Hidden, "Envelope was not archived together with category")
}

func (suite *TestSuiteStandard) TestCategorySetEnvelopes() {
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: suite.createTestBudget(models.BudgetCreate{}).ID,
	})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.ID})

	// Verify no envelopes are set
	assert.Len(suite.T(), category.Envelopes, 0)

	// Set envelopes and verify
	err := category.SetEnvelopes(suite.db)
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), category.Envelopes, 1)
	assert.Equal(suite.T(), envelope.ID, category.Envelopes[0].ID)
}

func (suite *TestSuiteStandard) TestCategorySelf() {
	assert.Equal(suite.T(), "Category", models.Category{}.Self())
}
