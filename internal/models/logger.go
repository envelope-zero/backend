package models

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	gorm_logger "gorm.io/gorm/logger"
)

type logger struct {
	Logger zerolog.Logger
}

func (l *logger) LogMode(gorm_logger.LogLevel) gorm_logger.Interface {
	return l
}

func (l *logger) Info(_ context.Context, s string, args ...interface{}) {
	l.Logger.Info().Msgf(s, args...)
}

func (l *logger) Warn(_ context.Context, s string, args ...interface{}) {
	l.Logger.Warn().Msgf(s, args...)
}

func (l *logger) Error(_ context.Context, s string, args ...interface{}) {
	l.Logger.Error().Msgf(s, args...)
}

func (l *logger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, _ := fc()
	fields := map[string]interface{}{
		"sql":      sql,
		"duration": elapsed,
	}

	if err != nil && !errors.Is(err, ErrResourceNotFound) {
		l.Logger.Error().Err(err).Fields(fields).Msg("[GORM] query error")
		return
	}

	l.Logger.Debug().Fields(fields).Msgf("[GORM] query")
}
