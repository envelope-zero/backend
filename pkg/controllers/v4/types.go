package v4

import (
	"time"

	ez_uuid "github.com/envelope-zero/backend/v5/internal/uuid"
)

type URITime struct {
	Time time.Time `uri:"time" example:"2024-01-07T18:43:00.271152Z"`
}

type URIMonth struct {
	Month time.Time `uri:"month" time_format:"2006-01" time_utc:"1" example:"2013-11" binding:"required"` // Year and month in YYYY-MM format
}

type URIID struct {
	ID ez_uuid.UUID `uri:"id" binding:"required"` // ID of the resource
}

type QueryMonth struct {
	Month time.Time `form:"month" time_format:"2006-01" time_utc:"1" example:"2022-07"` // Year and month in YYYY-MM format
}
