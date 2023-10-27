package models_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestMonthConfigSelf() {
	assert.Equal(suite.T(), "Month Config", models.MonthConfig{}.Self())
}
