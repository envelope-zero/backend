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
	Count  int   `json:"count"`  // The amount of records returned in this response
	Offset uint  `json:"offset"` // The offset for the first record returned
	Limit  int   `json:"limit"`  // The maximum amount of resources to return for this request
	Total  int64 `json:"total"`  // The total number of resources matching the query
}
