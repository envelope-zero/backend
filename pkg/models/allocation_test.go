package models_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestAllocationSelf() {
	assert.Equal(suite.T(), "Allocation", models.Allocation{}.Self())
}
