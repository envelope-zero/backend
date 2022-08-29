package models_test

import (
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteEnv) TestMigrateDatabase() {
	err := models.MigrateDatabase()
	assert.Nil(suite.T(), err)
}
