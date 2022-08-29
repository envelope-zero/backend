package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	gorm_zerolog "github.com/wei840222/gorm-zerolog"
	"gorm.io/gorm"
)

// DB is the database used by the backend.
var DB *gorm.DB

// ConnectDatabase connects to the database DB.
func ConnectDatabase(dialector func(string) gorm.Dialector, dsn string) error {
	var err error
	var db *gorm.DB

	config := &gorm.Config{
		// Set generated timestamps in UTC
		NowFunc: func() time.Time {
			return time.Now().In(time.UTC)
		},
		Logger: gorm_zerolog.New(),
	}

	db, err = gorm.Open(dialector(dsn), config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Get new connections after one hour
	sqlDB.SetConnMaxLifetime(time.Hour)

	// This is done to prevent SQLITE_BUSY errors.
	// If you have ideas how to improve this, you are very welcome to open an issue or a PR. Thank you!
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)

	DB = db

	return nil
}

// Database sets up the database.
func Database() error {
	// Create data directory
	dataDir := filepath.Join(".", "data")
	err := os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Connect to the database
	err = ConnectDatabase(sqlite.Open, "data/gorm.db?_pragma=foreign_keys(1)")
	if err != nil {
		return err
	}

	return nil
}
