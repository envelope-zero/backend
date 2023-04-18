package helpers

import (
	"crypto/sha256"
	"fmt"
)

// Sha256String calculates the SHA256 hash of a given string and returns its string representation.
func Sha256String(input string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(input)))
}
