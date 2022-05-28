package controllers

import (
	"time"
)

// This file holds types that are used over multiple files.

// Month is used to parse requests for data about a specific month.
type URIMonth struct {
	Month time.Time `uri:"month" time_format:"2006-01" time_utc:"1"`
}

type QueryFilter struct {
	Name string `form:"name"`
	Note string `form:"note"`
}
