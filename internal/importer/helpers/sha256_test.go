package helpers_test

import (
	"testing"

	"github.com/envelope-zero/backend/v5/internal/importer/helpers"
	"github.com/stretchr/testify/assert"
)

func TestSha256(t *testing.T) {
	s := helpers.Sha256String("Envelope Zero")
	assert.Equal(t, "dbac4a4ba50e42b6e04b43c2c9b3619e3668dc0a8caf050b584bdafaebee1787", s, "SHA256 checksum calculation is wrong!")
}
