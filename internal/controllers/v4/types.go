package v4

import (
	"time"

	ez_uuid "github.com/envelope-zero/backend/v7/internal/uuid"
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
