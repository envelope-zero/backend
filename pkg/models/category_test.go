package models_test

import (
	"strings"

	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestCategoryTrimWhitespace() {
	name := "\t Whitespace galore!   "
	note := " Some more whitespace in the notes    "

	category := suite.createTestCategory(models.CategoryCreate{
		Name:     name,
		Note:     note,
		BudgetID: suite.createTestBudget(models.Budget{}).ID,
	})

	assert.Equal(suite.T(), strings.TrimSpace(name), category.Name)
	assert.Equal(suite.T(), strings.TrimSpace(note), category.Note)
}

func (suite *TestSuiteStandard) TestCategoryArchiveArchivesEnvelopes() {
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: suite.createTestBudget(models.Budget{}).ID,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		CategoryID: category.ID,
		Archived:   false,
	})
	assert.False(suite.T(), envelope.Archived, "Envelope archived on creation, it should not be")

	// Archive the category
	err := models.DB.Model(&category).Select("Archived").Updates(models.Category{CategoryCreate: models.CategoryCreate{Archived: true}}).Error
	assert.Nil(suite.T(), err)

	// Verify that the envelope is archived
	err = models.DB.First(&envelope, envelope.ID).Error
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelope.Archived, "Envelope was not archived together with category")
}

func (suite *TestSuiteStandard) TestCategoryArchiveNoEnvelopes() {
	budget := suite.createTestBudget(models.Budget{})
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
		Name:     "TestCategoryArchiveNoEnvelopes",
	})

	category2 := suite.createTestCategory(models.CategoryCreate{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		CategoryID: category2.ID,
		Archived:   false,
	})
	assert.False(suite.T(), envelope.Archived, "Envelope archived on creation, it should not be")

	// Archive the empty category
	err := models.DB.Model(&category).Select("Archived").Updates(models.Category{CategoryCreate: models.CategoryCreate{Archived: true}}).Error
	assert.Nil(suite.T(), err)

	// Verify that the envelope is not archived
	err = models.DB.First(&envelope, envelope.ID).Error
	assert.Nil(suite.T(), err)
	assert.False(suite.T(), envelope.Archived, "Envelope was archived together with category")
}

func (suite *TestSuiteStandard) TestCategorySetEnvelopes() {
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: suite.createTestBudget(models.Budget{}).ID,
	})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.ID})

	// Set envelopes and verify
	envelopes, err := category.Envelopes(models.DB)
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), envelopes, 1)
	assert.Equal(suite.T(), envelope.ID, envelopes[0].ID)
}

func (suite *TestSuiteStandard) TestCategorySetEnvelopesDBFail() {
	category := suite.createTestCategory(models.CategoryCreate{
		BudgetID: suite.createTestBudget(models.Budget{}).ID,
	})
	suite.CloseDB()

	_, err := category.Envelopes(models.DB)
	assert.NotNil(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "database is closed")
}

func (suite *TestSuiteStandard) TestCategorySelf() {
	assert.Equal(suite.T(), "Category", models.Category{}.Self())
}
