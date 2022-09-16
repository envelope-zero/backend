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

// Connect opens the SQLite database and configures the connection pool.
func Connect(dsn string) (*gorm.DB, error) {
	config := &gorm.Config{
		// Set generated timestamps in UTC
		NowFunc: func() time.Time {
			return time.Now().In(time.UTC)
		},
		Logger: gorm_zerolog.New(),
	}

	db, err := gorm.Open(sqlite.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database object: %w", err)
	}

	// Get new connections after one hour
	sqlDB.SetConnMaxLifetime(time.Hour)

	// This is done to prevent SQLITE_BUSY errors.
	// If you have ideas how to improve this, you are very welcome to open an issue or a PR. Thank you!
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)

	return db, nil
}

// CreateDir creates a directory relative to the local path.
func CreateDir(path string) error {
	dataDir := filepath.Join(".", path)

	err := os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Failed to create directory: %w", err)
	}
	return nil
}
