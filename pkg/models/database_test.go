package models_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/database"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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

func TestOverspendMigration(t *testing.T) {
	// Copy test database to a temporary file
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "overspend-handling.db")

	input, err := os.ReadFile("../../testdata/migrations/overspend-handling.db")
	if err != nil {
		t.Error("Could not read overspend handling test database")
	}
	err = os.WriteFile(dbFile, input, 0o644)
	if err != nil {
		t.Error("Could not create temporary copy for database")
	}

	// Connect to the database
	db, err := database.Connect(fmt.Sprintf("%s?_pragma=foreign_keys(1)", dbFile))
	if err != nil {
		t.Errorf("Database connection failed with: %#v", err)
	}

	err = models.Migrate(db)
	if err != nil {
		t.Errorf("Database migration failed with: %s", err)
	}

	// The envelope are hard-coded here because the test database file does not change
	tests := []struct {
		envelopeID string
		month      string
		allocation int
	}{
		{"c9b0fce7-d51b-4641-9b43-666fe295cb30", "2022-11-01 00:00:00+00:00", -10},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2022-12-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-01-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-02-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-03-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-04-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-05-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-06-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-07-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-08-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-09-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-10-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-11-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2023-12-01 00:00:00+00:00", -5},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2024-01-01 00:00:00+00:00", -5},
		{"d9a80290-cc75-4a00-ad1d-de9d0c40f814", "2022-11-01 00:00:00+00:00", -120},
		{"d9a80290-cc75-4a00-ad1d-de9d0c40f814", "2022-12-01 00:00:00+00:00", -120},
		{"3c0de838-ef14-4b2f-83e6-079ffa321a32", "2024-02-01 00:00:00+00:00", -5},
		{"d9a80290-cc75-4a00-ad1d-de9d0c40f814", "2023-01-01 00:00:00+00:00", -120},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s - %s", tt.envelopeID, tt.month), func(t *testing.T) {
			// Get the number of records matching the month config. This must always be 1
			var count int
			db.Raw("SELECT count(*) FROM month_configs WHERE envelope_id = ? AND month = ? AND allocation = ?", tt.envelopeID, tt.month, tt.allocation).Scan(&count)
			assert.Equal(t, 1, count)
		})
	}

	if db.Migrator().HasColumn(&models.MonthConfig{}, "overspend_mode") {
		t.Error("column overspend_mode has not been deleted")
	}
}
