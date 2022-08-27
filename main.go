package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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

	// General settings for logging
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if gin.IsDebugging() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = log.Output(output).With().Logger()

	databaseInit()

	r, err := router.Router()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	// Set the port to the env variable, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}

	// The following code is taken from https://github.com/gin-gonic/examples/blob/91fb0d925b3935d2c6e4d9336d78cf828925789d/graceful-shutdown/graceful-shutdown/notify-without-context/server.go
	// and has been modified to not wait for the
	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Str("event", "Error during startup").Err(err).Msg("Router")
		}
	}()
	log.Info().Str("event", "Startup complete").Msg("Router")

	<-quit
	log.Info().Str("event", "Received SIGINT or SIGTERM, stopping gracefully with 25 seconds timeout").Msg("Router")

	// Create a context with a 25 second timeout for the server to shut down in
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Str("event", "Graceful shutdown failed, terminating").Err(err).Msg("Router")
	}
	log.Info().Str("event", "Backend stopped").Msg("Router")
}

// databaseInit initializes the data directory and database.
func databaseInit() {
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

	// Drop unused constraint in https://github.com/envelope-zero/backend/pull/274
	// Can be removed after the 1.0.0 release (we will require everyone to upgrade to 1.0.0 and then to further releases).
	err = database.DB.Migrator().DropConstraint(&models.Allocation{}, "month_valid")
	if err != nil {
		log.Debug().Err(err).Msg("Could not drop month_valid constraint on allocations")
	}

	// Migrate all models so that the schema is correct
	err = database.DB.AutoMigrate(models.Budget{}, models.Account{}, models.Category{}, models.Envelope{}, models.Transaction{}, models.Allocation{})
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}
