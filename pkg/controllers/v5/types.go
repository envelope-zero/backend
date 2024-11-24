package v5

import (
	"time"

	ez_uuid "github.com/envelope-zero/backend/v5/internal/uuid"
)

type URIMonth struct {
	URIID
	Month time.Time `uri:"month" time_format:"2006-01" time_utc:"1" example:"2013-11" binding:"required"` // Year and month in YYYY-MM format
}

type URIID struct {
	ID ez_uuid.UUID `uri:"id" binding:"required" format:"UUID"` // ID of the resource
}

type QueryMonth struct {
	Month time.Time `form:"month" time_format:"2006-01" time_utc:"1" example:"2022-07"` // Year and month in YYYY-MM format
}

// Pagination contains information about the pagination for collection endpoint responses.
type Pagination struct {
	Count  int   `json:"count" example:"25"`  // The amount of records returned in this response
	Offset uint  `json:"offset" example:"50"` // The offset for the first record returned
	Limit  int   `json:"limit" example:"25"`  // The maximum amount of resources to return for this request
	Total  int64 `json:"total" example:"827"` // The total number of resources matching the query
}
