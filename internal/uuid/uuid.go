package uuid

import (
	google_uuid "github.com/google/uuid"
)

type UUID struct {
	google_uuid.UUID
}

var Nil UUID

func New() UUID {
	return UUID{google_uuid.New()}
}

func NewString() string {
	return google_uuid.NewString()
}

// UnmarshalParam implements the uuid.Parse method
// from https://pkg.go.dev/github.com/google/uuid#Parse
// for UUID
func (u *UUID) UnmarshalParam(p string) error {
	if p == "" {
		*u = Nil
		return nil
	}

	parsed, e := google_uuid.Parse(p)
	if e != nil {
		return e
	}

	*u = UUID{parsed}
	return nil
}
