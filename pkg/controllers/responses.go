package controllers

import (
	"github.com/envelope-zero/backend/v2/pkg/models"
)

type ResponseTransactionV2 struct {
	Error string             `json:"error" example:"A human readable error message"` // This field contains a human readable error message
	Data  models.Transaction `json:"data"`                                           // This field contains the transaction data
}

type ResponseRenameRule struct {
	Error string            `json:"error" example:"A human readable error message"` // This field contains a human readable error message
	Data  models.RenameRule `json:"data"`                                           // This field contains the model data
}
