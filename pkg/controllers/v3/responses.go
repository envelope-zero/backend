package v3

// We use one type per Endpoint so that swagger can parse them - it cannot handle generics yet, see
// https://github.com/swaggo/swag/issues/1170

// Pagination contains information about the pagination for collection endpoint responses.
type Pagination struct {
	Count  int   `json:"count" example:"25"`  // The amount of records returned in this response
	Offset uint  `json:"offset" example:"50"` // The offset for the first record returned
	Limit  int   `json:"limit" example:"25"`  // The maximum amount of resources to return for this request
	Total  int64 `json:"total" example:"827"` // The total number of resources matching the query
}
