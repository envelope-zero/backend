package models_test

import (
	"github.com/envelope-zero/backend/pkg/models"
)

func (suite *TestSuiteClosedDB) TestMigrate() {
	err := models.Migrate(suite.db)
	suite.Assert().NotNil(err)
	suite.Assert().Contains(err.Error(), "error during DB migration")
}
