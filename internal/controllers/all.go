package controllers

import "time"

// This file holds types that are used over multiple files.

// Month is used to parse requests for data about a specific month.
type Month struct {
	Month time.Time `form:"month" time_format:"2006-01" time_utc:"1"`
}
