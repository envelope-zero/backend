package models_test

import (
	"strings"

	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func (suite *TestSuiteStandard) TestGoalSelf() {
	assert.Equal(suite.T(), "Goal", models.Goal{}.Self())
}

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
