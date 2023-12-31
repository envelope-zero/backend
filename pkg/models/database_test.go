package models_test

import (
	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/shopspring/decimal"
)

func (suite *TestSuiteStandard) TestMigrate() {
	suite.CloseDB()
	err := models.Migrate(suite.db)
	suite.Assert().NotNil(err)
	suite.Assert().Contains(err.Error(), "error during DB migration")
}

func (suite *TestSuiteStandard) TestMigrateWithExistingDB() {
	// Initialize the database to have all tables
	err := suite.db.AutoMigrate()
	suite.Assert().Nil(err, err)

	// Execute the migration again
	err = models.Migrate(suite.db)
	suite.Assert().Nil(err)
}

func (suite *TestSuiteStandard) TestMigrateAllocation() {
	err := suite.db.Raw("CREATE TABLE allocations (`id` text,`created_at` datetime,`updated_at` datetime,`deleted_at` datetime,`month` date,`amount` DECIMAL(20,8),`envelope_id` text,PRIMARY KEY (`id`))").Scan(nil).Error
	suite.Assert().Nil(err)

	err = suite.db.Raw("INSERT INTO allocations (id, envelope_id, month, amount) VALUES ('3afd1b7f-6bae-4256-aa78-89ef5dac7775', '41efaa99-1737-4dc6-818b-5d5f2ac65138', '2023-12-01 00:00:00+00:00', '10')").Scan(nil).Error
	suite.Assert().Nil(err)

	err = models.Migrate(suite.db)
	suite.Assert().Nil(err)

	type monthConfig struct {
		EnvelopeID string          `gorm:"column:envelope_id"`
		Month      types.Month     `gorm:"column:month"`
		Allocation decimal.Decimal `gorm:"column:allocation"`
	}

	var monthConfigs []monthConfig
	err = suite.db.Raw("SELECT envelope_id, month, allocation FROM month_configs WHERE envelope_id = '41efaa99-1737-4dc6-818b-5d5f2ac65138'").Scan(&monthConfigs).Error
	suite.Assert().Nil(err)
	suite.Assert().Len(monthConfigs, 1)
	suite.Assert().True(monthConfigs[0].Allocation.Equal(decimal.NewFromFloat(10)))

	var count int
	err = suite.db.Raw("SELECT count(name) FROM sqlite_master WHERE type='table' AND name='allocations'").Scan(&count).Error
	suite.Assert().Nil(err)
	suite.Assert().Equal(0, count)
}
