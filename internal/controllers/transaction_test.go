package controllers_test

import (
	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
)

type TransactionListResponse struct {
	test.APIResponse
	Data []models.Account
}
