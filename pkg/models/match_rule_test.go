package models_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestMatchRuleSelf() {
	assert.Equal(suite.T(), "Match Rule", models.MatchRule{}.Self())
}
