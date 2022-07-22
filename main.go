package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/internal/router"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// gin uses debug as the default mode, we use release for
	// security reasons
	ginMode, ok := os.LookupEnv("GIN_MODE")
	if !ok {
		gin.SetMode("release")
	} else {
		gin.SetMode(ginMode)
	}

	// Log format can be explicitly set.
	// If it is not set, it defaults to human readable for development
	// and JSON for release
	logFormat, ok := os.LookupEnv("LOG_FORMAT")
	output := io.Writer(os.Stdout)
	if (!ok && gin.IsDebugging()) || (ok && logFormat == "human") {
		output = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if gin.IsDebugging() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = log.Output(output).With().Timestamp().Logger()

	// Check with database driver to use. If DB_HOST is set, assume postgresql
	_, ok = os.LookupEnv("DB_HOST")

	var dsn string
	var dialector func(dsn string) gorm.Dialector
	if ok {
		log.Debug().Msg("DB_HOST is set, using postgresql")
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
		dialector = postgres.Open
	} else {
		log.Debug().Msg("DB_HOST is not set, using sqlite database")

		dataDir := filepath.Join(".", "data")
		err := os.MkdirAll(dataDir, os.ModePerm)
		if err != nil {
			panic("Could not create data directory")
		}

		dsn = "data/gorm.db?_pragma=foreign_keys(1)"
		dialector = sqlite.Open
	}

	// Connect to the database
	err := database.ConnectDatabase(dialector, dsn)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	// Migrate all models so that the schema is correct
	err = database.DB.AutoMigrate(models.Budget{}, models.Account{}, models.Category{}, models.Envelope{}, models.Transaction{}, models.Allocation{})
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	r, err := router.Router()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if err := r.Run(); err != nil {
		log.Fatal().Msg(err.Error())
	}
}
