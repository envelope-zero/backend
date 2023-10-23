package models_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestAllocationSelf() {
	assert.Equal(suite.T(), "Allocation", models.Allocation{}.Self())
}

func (suite *TestSuiteStandard) TestAllocationZero() {
	a := models.Allocation{
		AllocationCreate: models.AllocationCreate{
			Amount: decimal.Zero,
		},
	}

	err := a.BeforeSave(suite.db)
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), "allocation amounts must be non-zero. Instead of setting to zero, delete the Allocation", err.Error())
}
