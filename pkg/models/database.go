package models

import (
	"fmt"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	gorm_zerolog "github.com/wei840222/gorm-zerolog"
	"gorm.io/gorm"
)

var DB *gorm.DB

type EZContext string

const (
	DBContextURL EZContext = "ez-backend-url"
)

// Connect opens the SQLite database and configures the connection pool.
func Connect(dsn string) error {
	config := &gorm.Config{
		// Set generated timestamps in UTC
		NowFunc: func() time.Time {
			return time.Now().In(time.UTC)
		},
		Logger: gorm_zerolog.New(),
	}

	db, err := gorm.Open(sqlite.Open(dsn), config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database object: %w", err)
	}

	// Get new connections after one hour
	sqlDB.SetConnMaxLifetime(time.Hour)

	// This is done to prevent SQLITE_BUSY errors.
	// If you have ideas how to improve this, you are very welcome to open an issue or a PR. Thank you!
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)

	err = migrate(db)
	if err != nil {
		return err
	}

	// Set the exported variable
	DB = db

	return nil
}

// migrate migrates all models to the schema defined in the code.
func migrate(db *gorm.DB) (err error) {
	// https://github.com/envelope-zero/backend/issues/871
	// Remove with 5.0.0
	if db.Migrator().HasColumn(&Account{}, "hidden") {
		err = db.Migrator().RenameColumn(&Account{}, "Hidden", "Archived")
		if err != nil {
			return fmt.Errorf("error when renaming Hidden -> Archived for Account: %w", err)
		}
	}

	// https://github.com/envelope-zero/backend/issues/871
	// Remove with 5.0.0
	if db.Migrator().HasColumn(&Category{}, "hidden") {
		err = db.Migrator().RenameColumn(&Category{}, "Hidden", "Archived")
		if err != nil {
			return fmt.Errorf("error when renaming Hidden -> Archived for Category: %w", err)
		}
	}

	// https://github.com/envelope-zero/backend/issues/871
	// Remove with 5.0.0
	if db.Migrator().HasColumn(&Envelope{}, "hidden") {
		err = db.Migrator().RenameColumn(&Envelope{}, "Hidden", "Archived")
		if err != nil {
			return fmt.Errorf("error when renaming Hidden -> Archived for Envelope: %w", err)
		}
	}

	err = db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, MonthConfig{}, MatchRule{}, Goal{})
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
	}

	// https://github.com/envelope-zero/backend/issues/440
	// Remove with 5.0.0
	//
	// This migration has to be executed before the overspend handling migration
	// so that the allocation values are correct when updated by the overspend
	// handling migration
	if db.Migrator().HasTable("allocations") {
		err = migrateAllocationToMonthConfig(db)
		if err != nil {
			return fmt.Errorf("error during migrateAllocationToMonthConfig: %w", err)
		}
	}

	// https://github.com/envelope-zero/backend/issues/856
	// Remove with 5.0.0
	if db.Migrator().HasColumn(&MonthConfig{}, "overspend_mode") {
		err = migrateOverspendHandling(db)
		if err != nil {
			return fmt.Errorf("error during overspend handling migration: %w", err)
		}
	}

	// https://github.com/envelope-zero/backend/issues/359
	// Remove with 5.0.0
	if db.Migrator().HasColumn(&Transaction{}, "reconciled") {
		err = db.Migrator().DropColumn(&Transaction{}, "Reconciled")
		if err != nil {
			return fmt.Errorf("error when dropping reconciled column for transactions: %w", err)
		}
	}

	return nil
}

func migrateAllocationToMonthConfig(db *gorm.DB) (err error) {
	type allocation struct {
		EnvelopeID string          `gorm:"column:envelope_id"`
		Month      types.Month     `gorm:"column:month"`
		Amount     decimal.Decimal `gorm:"column:amount"`
	}

	var allocations []allocation
	err = db.Raw("select envelope_id, month, amount from allocations").Scan(&allocations).Error
	if err != nil {
		return err
	}

	// Execute all updates in a transaction
	tx := db.Begin()

	// For each allocation, read the values and update the MonthConfig with it
	for _, allocation := range allocations {
		id, err := uuid.Parse(allocation.EnvelopeID)
		if err != nil {
			tx.Rollback()
			return err
		}

		err = tx.Where(MonthConfig{
			Month:      allocation.Month,
			EnvelopeID: id,
		}).Assign(MonthConfig{MonthConfigCreate: MonthConfigCreate{
			Allocation: allocation.Amount,
		}}).FirstOrCreate(&MonthConfig{}).Error

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Raw("DROP TABLE allocations").Scan(nil).Error
	if err != nil {
		return err
	}

	tx.Commit()
	return nil
}

func migrateOverspendHandling(db *gorm.DB) (err error) {
	type overspend struct {
		EnvelopeID    string // `gorm:"column:envelope_id"`
		Month         types.Month
		OverspendMode string
	}

	var overspends []overspend
	err = db.Raw("select envelope_id, month, overspend_mode from month_configs WHERE overspend_mode = 'AFFECT_ENVELOPE'").Scan(&overspends).Error
	if err != nil {
		return err
	}

	// Execute all updates in a transaction
	tx := db.Begin()

	// For each overspend configuration, migrate the config as needed
	for _, overspend := range overspends {
		envelopeID, err := uuid.Parse(overspend.EnvelopeID)
		if err != nil {
			tx.Rollback()
			return err
		}

		var envelope Envelope
		err = tx.First(&envelope, envelopeID).Error
		if err != nil {
			tx.Rollback()
			return err
		}

		balance, err := envelope.Balance(tx, overspend.Month)
		if err != nil {
			tx.Rollback()
			return err
		}

		// If the envelope is not overspent (i.e. balance is >= 0), we don't need to do anything
		if balance.GreaterThanOrEqual(decimal.Zero) {
			continue
		}

		var monthConfig MonthConfig
		err = tx.Where(MonthConfig{
			Month:      overspend.Month.AddDate(0, 1),
			EnvelopeID: envelopeID,
		}).FirstOrCreate(&monthConfig).Error
		if err != nil {
			tx.Rollback()
			return err
		}

		// Add the balance
		// We need to subtract the overspent amount, since the balance is negative the overspent amount, we add it
		monthConfig.Allocation = monthConfig.Allocation.Add(balance)
		err = tx.Save(&monthConfig).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()

	return db.Migrator().DropColumn(&MonthConfig{}, "overspend_mode")
}
