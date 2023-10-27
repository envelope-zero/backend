package controllers

// We use one type per Endpoint so that swagger can parse them - it cannot handle generics yet, see
// https://github.com/swaggo/swag/issues/1170

type ResponseTransactionV2 struct {
	Error string        `json:"error" example:"A human readable error message"` // This field contains a human readable error message
	Data  TransactionV2 `json:"data"`                                           // This field contains the Transaction data
}

type ResponseMatchRule struct {
	Error string    `json:"error" example:"A human readable error message"` // This field contains a human readable error message
	Data  MatchRule `json:"data"`                                           // This field contains the MatchRule data
}
