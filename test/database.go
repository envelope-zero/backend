package test

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

// TmpFile returns the path to a unique file to be used in tests
func TmpFile(t *testing.T) string {
	dir := t.TempDir()
	return filepath.Join(dir, uuid.New().String())
}
