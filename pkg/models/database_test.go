package models_test

import (
	"github.com/envelope-zero/backend/pkg/models"
)

func (suite *TestSuiteStandard) TestMigrate() {
	suite.DisconnectDB()
	err := models.Migrate(suite.db)
	suite.Assert().NotNil(err)
	suite.Assert().Contains(err.Error(), "error during DB migration")
}
