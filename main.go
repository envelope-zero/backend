package main

import (
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

	// Create data directory
	dataDir := filepath.Join(".", "data")
	err := os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	// Connect to the database
	err = database.ConnectDatabase(sqlite.Open, "data/gorm.db?_pragma=foreign_keys(1)")
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
