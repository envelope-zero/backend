package models_test

import (
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteEnv) TestRawTransactions() {
	_, err := models.RawTransactions("INVALID query string")
	assert.NotNil(suite.T(), err)
}
