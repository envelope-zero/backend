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

	// Verify that the envelope is archived
	err = suite.db.First(&envelope, envelope.ID).Error
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelope.Hidden, "Envelope was not archived together with category")
}

func (suite *TestSuiteStandard) TestCategoryArchiveNoEnvelopes() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
		Name:     "TestCategoryArchiveNoEnvelopes",
	})

	category2 := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		CategoryID: category2.ID,
		Hidden:     false,
	})
	assert.False(suite.T(), envelope.Hidden, "Envelope archived on creation, it should not be")

	// Archive the empty category
	err := suite.db.Model(&category).Select("Hidden").Updates(models.Category{CategoryCreate: models.CategoryCreate{Hidden: true}}).Error
	assert.Nil(suite.T(), err)

	// Verify that the envelope is not archived
	err = suite.db.First(&envelope, envelope.ID).Error
	assert.Nil(suite.T(), err)
	assert.False(suite.T(), envelope.Hidden, "Envelope was archived together with category")
}

func (suite *TestSuiteStandard) TestCategorySetEnvelopes() {
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: suite.createTestBudget(models.BudgetCreate{}).ID,
	})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.ID})

	// Set envelopes and verify
	envelopes, err := category.Envelopes(suite.db)
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), envelopes, 1)
	assert.Equal(suite.T(), envelope.ID, envelopes[0].ID)
}

func (suite *TestSuiteStandard) TestCategorySetEnvelopesDBFail() {
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: suite.createTestBudget(models.BudgetCreate{}).ID,
	})
	suite.CloseDB()

	_, err := category.Envelopes(suite.db)
	assert.NotNil(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "database is closed")
}

func (suite *TestSuiteStandard) TestCategorySelf() {
	assert.Equal(suite.T(), "Category", models.Category{}.Self())
}
