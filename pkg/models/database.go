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
	// https://github.com/envelope-zero/backend/issues/874
	// Remove with 6.0.0
	if db.Migrator().HasColumn(&Transaction{}, "budget_id") {
		err := db.Migrator().DropConstraint(&Transaction{}, "fk_transactions_budget")
		if err != nil {
			return fmt.Errorf("error when dropping BudgetID column for Transaction: %w", err)
		}

		err = db.Migrator().DropColumn(&Transaction{}, "budget_id")
		if err != nil {
			return fmt.Errorf("error when dropping BudgetID column for Transaction: %w", err)
		}
	}

	err = db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, MonthConfig{}, MatchRule{}, Goal{})
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
	}

	return nil
}
