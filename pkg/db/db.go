package db

import "github.com/envelope-zero/backend/v5/internal/models"

func Connect(dsn string) error {
	return models.Connect(dsn)
}
