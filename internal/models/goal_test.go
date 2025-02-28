package models_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func (suite *TestSuiteStandard) TestGoalAfterSave() {
	tests := []struct {
		amount decimal.Decimal
		err    error
	}{
		{decimal.NewFromFloat(-10), models.ErrGoalAmountNotPositive},
		{decimal.NewFromFloat(750), nil},
	}

	for _, tt := range tests {
		g := models.Goal{
			Amount: tt.amount,
		}

		err := g.AfterSave(&gorm.DB{})
		assert.Equal(suite.T(), tt.err, err)
	}
}

func (suite *TestSuiteStandard) TestGoalTrimWhitespace() {
	budget := suite.createTestBudget(models.Budget{})
	category := suite.createTestCategory(models.Category{BudgetID: budget.ID})
	envelope := suite.createTestEnvelope(models.Envelope{CategoryID: category.ID})

	note := " Whitespace    "
	name := "  There is whitespace here  \t"

	goal := suite.createTestGoal(models.Goal{
		EnvelopeID: envelope.ID,
		Amount:     decimal.NewFromFloat(100),
		Name:       name,
		Note:       note,
	})

	assert.Equal(suite.T(), strings.TrimSpace(name), goal.Name)
	assert.Equal(suite.T(), strings.TrimSpace(note), goal.Note)
}

func (suite *TestSuiteStandard) TestGoalUpdate() {
	budget := suite.createTestBudget(models.Budget{})
	category := suite.createTestCategory(models.Category{BudgetID: budget.ID})
	envelope := suite.createTestEnvelope(models.Envelope{CategoryID: category.ID, Name: "TestGoalUpdate"})

	goal := suite.createTestGoal(models.Goal{
		EnvelopeID: envelope.ID,
		Amount:     decimal.NewFromFloat(100),
	})

	tests := []struct {
		name       string
		envelopeID uuid.UUID
		err        error
	}{
		{
			"Valid envelope ID",
			suite.createTestEnvelope(models.Envelope{CategoryID: category.ID}).ID,
			nil,
		},
		{
			"Invalid envelope ID",
			uuid.New(),
			models.ErrResourceNotFound,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			update := models.Goal{
				EnvelopeID: tt.envelopeID,
			}
			err := models.DB.Model(&goal).Updates(update).Error
			assert.ErrorIs(t, err, tt.err, "Error is: %s", err)
		})
	}
}

func (suite *TestSuiteStandard) TestGoalExport() {
	t := suite.T()

	budget := suite.createTestBudget(models.Budget{})
	category := suite.createTestCategory(models.Category{BudgetID: budget.ID})
	envelope := suite.createTestEnvelope(models.Envelope{CategoryID: category.ID})

	for i := range 2 {
		_ = suite.createTestGoal(models.Goal{EnvelopeID: envelope.ID, Name: fmt.Sprint(i), Amount: decimal.NewFromFloat(17)})
	}

	raw, err := models.Goal{}.Export()
	if err != nil {
		require.Fail(t, "goal export failed", err)
	}

	var goals []models.Goal
	err = json.Unmarshal(raw, &goals)
	if err != nil {
		require.Fail(t, "JSON could not be unmarshaled", err)
	}

	require.Len(t, goals, 2, "Number of goals in export is wrong")
}
