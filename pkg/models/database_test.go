package models_test

import (
	"github.com/envelope-zero/backend/v2/pkg/models"
)

func (suite *TestSuiteStandard) TestMigrate() {
	suite.CloseDB()
	err := models.Migrate(suite.db)
	suite.Assert().NotNil(err)
	suite.Assert().Contains(err.Error(), "error during DB migration")
}
