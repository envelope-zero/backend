package controllers

import (
	"errors"
	"strings"

	"github.com/envelope-zero/backend/v2/pkg/httperrors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// TryDBConnect checks the database error to decide
// if a reconnection attempt makes sense and executes
// the reconnection.
// It returns nil for success and an error if reconnecting
// the database is not possible or not a sensible decision
//
// In cases where the database file might have been deleted,
// we err on the side of caution and do *not* reconnect the
// database so that we do not have two states that need to
// be merged manually by the user afterwards
//
// TODO: Actually implement the reconnection logic. This is
// not yet done as we currently do not store.
func TryDBConnect(e error) error {
	// If there is no error or the error is a closed database, we try to connect
	// to the database.
	//
	// A closed database will happen if the database is not initialized correctly
	// or there was an error on the intialization
	if e == nil || strings.Contains(e.Error(), "sql: database is closed") {
		log.Warn().Str("db", "database is closed, please restart the backend").Msg("TryDBConnect")
		return errors.New("database is closed")

		// TODO: Implement something like the following for the reconenction
		// log.Warn().Str("db", "(re)connecting database").Msg("TryDBConnect")
		// err := database.Database()
		// if err != nil {
		// 	return err
		// }

		// return nil
	}

	// We do not try to connect to the database here. This is due to the database
	// file possibly having been deleted
	if strings.Contains(e.Error(), "attempt to write a readonly database (1032)") {
		log.Error().Str("db", "database is read-only, not attempting to reconnect. Verify that the database file has not been deleted and restart the backend.").Msg("TryDBConnect")
		return errors.New("database is read-only")
	}

	if e != gorm.ErrRecordNotFound {
		log.Info().Err(e).Msg("Database")
	}
	return e
}

// queryWithRetry tries to execute a query. If it fails, it
// tries to reconnect the database and retries the query once.
func queryWithRetry(c *gin.Context, tx *gorm.DB, notFoundMsg ...string) bool {
	err := tx.Error
	if err != nil {
		if TryDBConnect(err) != nil {
			httperrors.Handler(c, err, notFoundMsg...)
			return false
		}

		err = tx.Error
		if err != nil {
			httperrors.Handler(c, err, notFoundMsg...)
			return false
		}
	}

	return true
}
