package models

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	go_sqlite "github.com/glebarez/go-sqlite"
	"github.com/glebarez/sqlite"
	"github.com/rs/zerolog/log"
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
		Logger: &logger{
			Logger: log.Logger,
		},
	}

	// Migration with foreign keys disabled since we're dropping tables
	// during migration
	//
	// sqlite does not support ALTER COLUMN, so tables are copied to a temporary table,
	// then the table is dropped and recreated
	db, err := gorm.Open(sqlite.Open(dsn), config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	err = migrate(db)
	if err != nil {
		return err
	}

	// Close the connection
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database object: %w", err)
	}
	sqlDB.Close()

	// Now, reconnect with foreign keys enabled
	dsn = fmt.Sprintf("%s?_pragma=foreign_keys(1)", dsn)
	db, err = gorm.Open(sqlite.Open(dsn), config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err = db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database object: %w", err)
	}

	// Get new connections after one hour
	sqlDB.SetConnMaxLifetime(time.Hour)

	// This is done to prevent SQLITE_BUSY errors.
	// If you have ideas how to improve this, you are very welcome to open an issue or a PR. Thank you!
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)

	// Query callbacks
	err = db.Callback().Query().After("*").Register("envelope_zero:after_query", queryCallback)
	if err != nil {
		return err
	}

	err = db.Callback().Query().After("*").Register("envelope_zero:after_query_general", generalCallback)
	if err != nil {
		return err
	}

	// Create callbacks
	err = db.Callback().Create().After("*").Register("envelope_zero:after_create", createUpdateCallback)
	if err != nil {
		return err
	}

	err = db.Callback().Create().After("*").Register("envelope_zero:after_create_general", generalCallback)
	if err != nil {
		return err
	}

	// Update callbacks
	err = db.Callback().Update().After("*").Register("envelope_zero:after_update", createUpdateCallback)
	if err != nil {
		return err
	}

	err = db.Callback().Update().After("*").Register("envelope_zero:after_update_general", generalCallback)
	if err != nil {
		return err
	}

	// Delete callbacks
	err = db.Callback().Delete().After("*").Register("envelope_zero:after_delete_general", generalCallback)
	if err != nil {
		return err
	}

	// Set the exported variable
	DB = db

	return nil
}

// queryCallback replaces the generic "no record" error with a more user
// friendly one
func queryCallback(db *gorm.DB) {
	if errors.Is(db.Error, gorm.ErrRecordNotFound) {
		// Use the table name as information about the type of resource
		// and replace "_" with "[space]"
		name := strings.ReplaceAll(db.Statement.Table, "_", " ")

		// Replace pluralized "ies" with "y"
		match := regexp.MustCompile("ies$")
		name = match.ReplaceAllString(name, "y")

		// Remove plural "s"
		name = strings.TrimRight(name, "s")

		db.Error = fmt.Errorf("%w %s matching your query", ErrResourceNotFound, name)
	}
}

// createUpdateCallback inspects errors returned by the database for create
// and update calls and replaces them with user friendly ones
func createUpdateCallback(db *gorm.DB) {
	if db.Error == nil {
		return
	}

	// Account name must be unique per Budget
	if strings.Contains(db.Error.Error(), "UNIQUE constraint failed: accounts.budget_id, accounts.name") {
		db.Error = ErrAccountNameNotUnique
	}

	// Category names need to be unique per budget
	if strings.Contains(db.Error.Error(), "UNIQUE constraint failed: categories.budget_id, categories.name") {
		db.Error = ErrCategoryNameNotUnique
	}

	// Unique envelope names per category
	if strings.Contains(db.Error.Error(), "UNIQUE constraint failed: envelopes.category_id, envelopes.name") {
		db.Error = ErrEnvelopeNameNotUnique
	}

	if strings.Contains(db.Error.Error(), "UNIQUE constraint failed: month_configs.envelope_id, month_configs.month") {
		db.Error = ErrMonthConfigMonthNotUnique
	}

	// Source and destination accounts need to be different
	if strings.Contains(db.Error.Error(), "CHECK constraint failed: source_destination_different") {
		db.Error = ErrSourceDoesNotEqualDestination
	}
}

// generalCallback handles unspecified errors.
//
// For these errors, we cannot provide the user with a helpful message.
// Instead, the error is logged and we return a general message to users.
func generalCallback(db *gorm.DB) {
	if db.Error == nil {
		return
	}

	// "sql: database is closed" is hard-coded in the sql module, see
	// https://cs.opensource.google/go/go/+/master:src/database/sql/sql.go;l=1298;drc=0d018b49e33b1383dc0ae5cc968e800dffeeaf7d
	if db.Error.Error() == "sql: database is closed" || reflect.TypeOf(db.Error) == reflect.TypeOf(&go_sqlite.Error{}) {
		// A general error where we cannot provide more useful information to the end user
		// We log the error and provide a general error message so that server admins can debug
		log.Error().Msgf("%T: %v", db.Error, db.Error.Error())
		db.Error = ErrGeneral

		return
	}
}

// migrate migrates all models to the schema defined in the code.
func migrate(db *gorm.DB) (err error) {
	err = removeDeletedAt(db)
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
	}

	err = db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, MonthConfig{}, MatchRule{}, Goal{})
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
	}

	return nil
}

// removeDeletedAt removes the DeletedAt column from all models, removing all deleted resources in the process
func removeDeletedAt(db *gorm.DB) (err error) {
	if db.Migrator().HasColumn(&Account{}, "deleted_at") {
		var accounts []Account
		err = db.Model(&Account{}).
			Where("deleted_at != ''").
			Find(&accounts).Error
		if err != nil {
			return fmt.Errorf("error when getting soft-deleted accounts: %w", err)
		}

		if len(accounts) > 0 {
			log.Info().Msgf("Migration: Deleting %d previously soft-deleted accounts", len(accounts))
			err = db.Delete(&accounts).Error
			if err != nil {
				return fmt.Errorf("error when deleting soft-deleted accounts: %w", err)
			}
		}

		err = db.Migrator().DropColumn(&Account{}, "deleted_at")
		if err != nil {
			return fmt.Errorf("error when dropping DeletedAt column for accounts: %w", err)
		}
	}

	if db.Migrator().HasColumn(&Budget{}, "deleted_at") {
		var budgets []Budget
		err = db.Model(&Budget{}).
			Where("deleted_at != ''").
			Find(&budgets).Error
		if err != nil {
			return fmt.Errorf("error when getting soft-deleted budgets: %w", err)
		}

		if len(budgets) > 0 {
			log.Info().Msgf("Migration: Deleting %d previously soft-deleted budgets", len(budgets))
			err = db.Delete(&budgets).Error
			if err != nil {
				return fmt.Errorf("error when deleting soft-deleted budgets: %w", err)
			}
		}

		err = db.Migrator().DropColumn(&Budget{}, "deleted_at")
		if err != nil {
			return fmt.Errorf("error when dropping DeletedAt column for budgets: %w", err)
		}
	}

	if db.Migrator().HasColumn(&Category{}, "deleted_at") {
		var categories []Category
		err = db.Model(&Category{}).
			Where("deleted_at != ''").
			Find(&categories).Error
		if err != nil {
			return fmt.Errorf("error when getting soft-deleted categories: %w", err)
		}

		if len(categories) > 0 {
			log.Info().Msgf("Migration: Deleting %d previously soft-deleted categories", len(categories))
			err = db.Delete(&categories).Error
			if err != nil {
				return fmt.Errorf("error when deleting soft-deleted categories: %w", err)
			}
		}

		err = db.Migrator().DropColumn(&Category{}, "deleted_at")
		if err != nil {
			return fmt.Errorf("error when dropping DeletedAt column for categories: %w", err)
		}
	}

	if db.Migrator().HasColumn(&Envelope{}, "deleted_at") {
		var envelopes []Envelope
		err = db.Model(&Envelope{}).
			Where("deleted_at != ''").
			Find(&envelopes).Error
		if err != nil {
			return fmt.Errorf("error when getting soft-deleted envelopes: %w", err)
		}

		if len(envelopes) > 0 {
			log.Info().Msgf("Migration: Deleting %d previously soft-deleted envelopes", len(envelopes))
			err = db.Delete(&envelopes).Error
			if err != nil {
				return fmt.Errorf("error when deleting soft-deleted envelopes: %w", err)
			}
		}

		err = db.Migrator().DropColumn(&Envelope{}, "deleted_at")
		if err != nil {
			return fmt.Errorf("error when dropping DeletedAt column for envelopes: %w", err)
		}
	}

	if db.Migrator().HasColumn(&Goal{}, "deleted_at") {
		var goals []Goal
		err = db.Model(&Goal{}).
			Where("deleted_at != ''").
			Find(&goals).Error
		if err != nil {
			return fmt.Errorf("error when getting soft-deleted goals: %w", err)
		}

		if len(goals) > 0 {
			log.Info().Msgf("Migration: Deleting %d previously soft-deleted goals", len(goals))
			err = db.Delete(&goals).Error
			if err != nil {
				return fmt.Errorf("error when deleting soft-deleted goals: %w", err)
			}
		}

		err = db.Migrator().DropColumn(&Goal{}, "deleted_at")
		if err != nil {
			return fmt.Errorf("error when dropping DeletedAt column for goals: %w", err)
		}
	}

	if db.Migrator().HasColumn(&MatchRule{}, "deleted_at") {
		var matchRules []MatchRule
		err = db.Model(&MatchRule{}).
			Where("deleted_at != ''").
			Find(&matchRules).Error
		if err != nil {
			return fmt.Errorf("error when getting soft-deleted matchRules: %w", err)
		}

		if len(matchRules) > 0 {
			log.Info().Msgf("Migration: Deleting %d previously soft-deleted matchRules", len(matchRules))
			err = db.Delete(&matchRules).Error
			if err != nil {
				return fmt.Errorf("error when deleting soft-deleted matchRules: %w", err)
			}
		}

		err = db.Migrator().DropColumn(&MatchRule{}, "deleted_at")
		if err != nil {
			return fmt.Errorf("error when dropping DeletedAt column for match rules: %w", err)
		}
	}

	if db.Migrator().HasColumn(&Transaction{}, "deleted_at") {
		var transactions []Transaction
		err = db.Model(&Transaction{}).
			Where("deleted_at != ''").
			Find(&transactions).Error
		if err != nil {
			return fmt.Errorf("error when getting soft-deleted transactions: %w", err)
		}

		if len(transactions) > 0 {
			log.Info().Msgf("Migration: Deleting %d previously soft-deleted transactions", len(transactions))
			err = db.Delete(&transactions).Error
			if err != nil {
				return fmt.Errorf("error when deleting soft-deleted transactions: %w", err)
			}
		}

		err = db.Migrator().DropColumn(&Transaction{}, "deleted_at")
		if err != nil {
			return fmt.Errorf("error when dropping DeletedAt column for Transaction: %w", err)
		}
	}

	return nil
}
