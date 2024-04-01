package models_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *TestSuiteStandard) TestCategoryTrimWhitespace() {
	name := "\t Whitespace galore!   "
	note := " Some more whitespace in the notes    "

	category := suite.createTestCategory(models.Category{
		Name:     name,
		Note:     note,
		BudgetID: suite.createTestBudget(models.Budget{}).ID,
	})

	assert.Equal(suite.T(), strings.TrimSpace(name), category.Name)
	assert.Equal(suite.T(), strings.TrimSpace(note), category.Note)
}

func (suite *TestSuiteStandard) TestCategoryArchiveArchivesEnvelopes() {
	category := suite.createTestCategory(models.Category{
		BudgetID: suite.createTestBudget(models.Budget{}).ID,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		CategoryID: category.ID,
		Archived:   false,
	})
	assert.False(suite.T(), envelope.Archived, "Envelope archived on creation, it should not be")

	// Archive the category
	err := models.DB.Model(&category).Select("Archived").Updates(models.Category{Archived: true}).Error
	assert.Nil(suite.T(), err)

	// Verify that the envelope is archived
	err = models.DB.First(&envelope, envelope.ID).Error
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), envelope.Archived, "Envelope was not archived together with category")
}

func (suite *TestSuiteStandard) TestCategoryArchiveNoEnvelopes() {
	budget := suite.createTestBudget(models.Budget{})
	category := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
		Name:     "TestCategoryArchiveNoEnvelopes",
	})

	category2 := suite.createTestCategory(models.Category{
		BudgetID: budget.ID,
	})

	envelope := suite.createTestEnvelope(models.Envelope{
		CategoryID: category2.ID,
		Archived:   false,
	})
	assert.False(suite.T(), envelope.Archived, "Envelope archived on creation, it should not be")

	// Archive the empty category
	err := models.DB.Model(&category).Select("Archived").Updates(models.Category{Archived: true}).Error
	assert.Nil(suite.T(), err)

	// Verify that the envelope is not archived
	err = models.DB.First(&envelope, envelope.ID).Error
	assert.Nil(suite.T(), err)
	assert.False(suite.T(), envelope.Archived, "Envelope was archived together with category")
}

func (suite *TestSuiteStandard) TestCategorySetEnvelopes() {
	category := suite.createTestCategory(models.Category{
		BudgetID: suite.createTestBudget(models.Budget{}).ID,
	})
	envelope := suite.createTestEnvelope(models.Envelope{CategoryID: category.ID})

	// Set envelopes and verify
	envelopes, err := category.Envelopes(models.DB)
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), envelopes, 1)
	assert.Equal(suite.T(), envelope.ID, envelopes[0].ID)
}

func (suite *TestSuiteStandard) TestCategorySetEnvelopesDBFail() {
	category := suite.createTestCategory(models.Category{
		BudgetID: suite.createTestBudget(models.Budget{}).ID,
	})
	suite.CloseDB()

	_, err := category.Envelopes(models.DB)
	suite.Assert().ErrorIs(err, models.ErrGeneral)
}

func (suite *TestSuiteStandard) TestCategoryExport() {
	t := suite.T()

	budget := suite.createTestBudget(models.Budget{})

	for i := range 2 {
		_ = suite.createTestCategory(models.Category{BudgetID: budget.ID, Name: fmt.Sprint(i)})
	}

	raw, err := models.Category{}.Export()
	if err != nil {
		require.Fail(t, "category export failed", err)
	}

	var categories []models.Category
	err = json.Unmarshal(raw, &categories)
	if err != nil {
		require.Fail(t, "JSON could not be unmarshaled", err)
	}

	require.Len(t, categories, 2, "Number of categories in export is wrong")
}
