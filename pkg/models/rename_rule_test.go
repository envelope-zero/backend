package models_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestRenameRuleSelf() {
	assert.Equal(suite.T(), "Rename Rule", models.RenameRule{}.Self())
}
