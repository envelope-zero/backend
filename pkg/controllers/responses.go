package controllers

// We use one type per Endpoint so that swagger can parse them - it cannot handle generics yet, see
// https://github.com/swaggo/swag/issues/1170

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
)

type ResponseTransactionV2 struct {
	Error string        `json:"error" example:"A human readable error message"` // This field contains a human readable error message
	Data  TransactionV2 `json:"data"`                                           // This field contains the transaction data
}

type ResponseMatchRule struct {
	Error string           `json:"error" example:"A human readable error message"` // This field contains a human readable error message
	Data  models.MatchRule `json:"data"`                                           // This field contains the model data
}
