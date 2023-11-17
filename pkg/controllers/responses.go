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

// Pagination contains information about the pagination for collection endpoint responses.
type Pagination struct {
	Count  int   `json:"count" example:"25"`  // The amount of records returned in this response
	Offset uint  `json:"offset" example:"50"` // The offset for the first record returned
	Limit  int   `json:"limit" example:"25"`  // The maximum amount of resources to return for this request
	Total  int64 `json:"total" example:"827"` // The total number of resources matching the query
}
