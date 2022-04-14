package models

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/glebarez/sqlite"
	gorm_zerolog "github.com/wei840222/gorm-zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB is the database used by the backend.
var DB *gorm.DB

// ConnectDatabase connects to the database DB.
func ConnectDatabase() error {
	var err error
	var db *gorm.DB

	config := &gorm.Config{
		// Set generated timestamps in UTC
		NowFunc: func() time.Time {
			return time.Now().In(time.UTC)
		},
		Logger: gorm_zerolog.New(),
	}

	// logger.New(
	// 	golog.New(os.Stdout, "\r\n", golog.LstdFlags),
	// 	logger.Config{
	// 		SlowThreshold:             time.Millisecond,
	// 		LogLevel:                  logger.Error,
	// 		IgnoreRecordNotFoundError: true,
	// 		Colorful:                  true,
	// 	},
	// ),

	// Check with database driver to use. If DB_HOST is set, assume postgresql
	_, ok := os.LookupEnv("DB_HOST")
	if ok {
		log.Debug().Msg("DB_HOST is set, using postgresql")
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
		db, err = gorm.Open(postgres.Open(dsn), config)
	} else {
		log.Debug().Msg("DB_HOST is not set, using sqlite database")

		dataDir := filepath.Join(".", "data")
		err = os.MkdirAll(dataDir, os.ModePerm)
		if err != nil {
			panic("Could not create data directory")
		}
		db, err = gorm.Open(sqlite.Open("data/gorm.db"), config)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, Allocation{})
	if err != nil {
		return err
	}

	DB = db
	return nil
}
