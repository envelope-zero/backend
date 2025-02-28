package db

import "github.com/envelope-zero/backend/v7/internal/models"

func Connect(dsn string) error {
	return models.Connect(dsn)
}
