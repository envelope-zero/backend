package uuid_test

import (
	"testing"

	"github.com/envelope-zero/backend/v7/internal/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNew tests that a new UUID can be generated.
// We don't validate the result, google/uuid already has tests
func TestNew(_ *testing.T) {
	_ = uuid.New()
}

// TestNewString tests that a new UUID can be generated as string.
// We don't validate the result, google/uuid already has tests
func TestNewString(_ *testing.T) {
	_ = uuid.NewString()
}

func TestUnmarshalParam(t *testing.T) {
	u := uuid.UUID{}

	// an invalid UUID does not parse
	assert.NotNil(t, u.UnmarshalParam("not a valid UUID"))

	// A valid UUID in a string parses
	id := uuid.NewString()
	assert.Nil(t, u.UnmarshalParam(id))
	assert.Equal(t, id, u.String())

	// Empty string parses to Nil UIID
	id = ""
	assert.Nil(t, u.UnmarshalParam(id))
	assert.Equal(t, uuid.Nil, u)
}
