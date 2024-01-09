package v4

import "time"

type URITime struct {
	Time time.Time `uri:"time" example:"2024-01-07T18:43:00.271152Z"`
}

type URIMonth struct {
	Month time.Time `uri:"month" time_format:"2006-01" time_utc:"1" example:"2013-11"` // Year and month
}

type QueryMonth struct {
	Month time.Time `form:"month" time_format:"2006-01" time_utc:"1" example:"2022-07"` // Year and month
}
