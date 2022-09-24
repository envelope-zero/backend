package ynab4_test

import (
	"os"
	"testing"

	"github.com/envelope-zero/backend/pkg/importer/parser/ynab4"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	f, err := os.OpenFile("../../../../testdata/Budget.yfull", os.O_RDONLY, 0o400)
	if err != nil {
		assert.FailNow(t, "Failed to open the test file", err)
	}

	_, err = ynab4.Parse(f)
	if err != nil {
		assert.Fail(t, "Parsing failed", err)
	}
}
