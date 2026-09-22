// Package types implements special types for Envelope Zero.
package types

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Month is a month in a specific year.
type Month time.Time

// NewMonth returns a new Month.
func NewMonth(year int, month time.Month) Month {
	return Month(time.Date(year, month, 1, 0, 0, 0, 0, time.UTC))
}

// String returns the time formatted as YYYY-MM.
func (m Month) String() string {
	return fmt.Sprintf("%04d-%02d", time.Time(m).Year(), time.Time(m).Month())
}

// MarshalJSON implements the json.Marshaler interface.
// The output is the result of m.StringRFC3339().
func (m Month) MarshalJSON() ([]byte, error) {
	return time.Time(m).MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The month is expected to be a string in a format accepted by ParseDate.
// From the parsed string, everything is then ignored except the year and month
func (m *Month) UnmarshalJSON(data []byte) error {
	value := strings.Trim(string(data), `"`) // get rid of "
	if value == "" || value == "null" {
		return nil
	}

	// This allows to parse strings in the "2006-01-02" format
	match, err := regexp.MatchString("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", string(value))
	if err != nil {
		return err
	}

	// This is the default pattern
	pattern := "2006-01-02T15:04:05Z07:00"
	if match {
		pattern = "2006-01-02"
	}

	t, err := time.Parse(pattern, string(value))
	if err != nil {
		return err
	}

	month := NewMonth(t.Year(), t.Month())
	*m = month
	return nil
}

// MonthOf returns the Month in which a time occurs in that time's location.
func MonthOf(t time.Time) Month {
	year, month, _ := t.Date()
	return Month(time.Date(year, month, 1, 0, 0, 0, 0, time.Time(t).Location()))
}

// ParseDateToMonth parses a string in RFC3339 full-date format and returns the Month value it represents.
func ParseDateToMonth(s string) (Month, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Month{}, err
	}

	return MonthOf(t), nil
}

// ParseMonth parses a "YYYY-MM" string and returns the Month value it represents
func ParseMonth(s string) (Month, error) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return Month{}, err
	}

	return MonthOf(t), nil
}

// Scan writes the value from the database.
func (m *Month) Scan(value interface{}) (err error) {
	nullTime := &sql.NullTime{}
	err = nullTime.Scan(value)
	*m = Month(nullTime.Time)
	return err
}

// Value returns the value for the SQL driver to write to the database.
func (m Month) Value() (driver.Value, error) {
	year, month, _ := time.Time(m).Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC), nil
}

// GormDataType defines the data type used by gorm the type.
func (Month) GormDataType() string {
	return "date"
}

// IsZero reports if the month is the zero value.
func (m Month) IsZero() bool {
	return time.Time(m).IsZero()
}

// AddDate adds a specified amount of years and months.
func (m Month) AddDate(years, months int) Month {
	return Month(time.Time(m).AddDate(years, months, 0))
}

// Before reports whether the month instant m is before n.
func (m Month) Before(n Month) bool {
	return time.Time(m).Before(time.Time(n))
}

// After reports whether the month instant m is after n.
func (m Month) After(n Month) bool {
	return time.Time(m).After(time.Time(n))
}

// BeforeTime reports whether the month instant m is before the time instant t.
func (m Month) BeforeTime(t time.Time) bool {
	return time.Time(m).Before(t)
}

// AfterTime reports whether the month instant m is after the time instant t.
func (m Month) AfterTime(t time.Time) bool {
	return time.Time(m).After(t)
}

// Equal reports whether m and n represent the same month.
func (m Month) Equal(n Month) bool {
	return time.Time(m).Equal(time.Time(n))
}

// Contains reports whether the time instant is in the month.
func (m Month) Contains(t time.Time) bool {
	return t.Year() == time.Time(m).Year() && t.Month() == time.Time(m).Month()
}
