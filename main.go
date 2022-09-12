package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/envelope-zero/backend/pkg/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/router"
	"github.com/gin-gonic/gin"
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

	// Connect to and migrate the database
	err := database.Database()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	err = models.MigrateDatabase()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	r, err := router.Config()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	// Attach the routes to the root URL
	router.AttachRoutes(r.Group("/"))

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
