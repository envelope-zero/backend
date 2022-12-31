package ynab4_test

import (
	"errors"
	"os"
	"testing"
	"testing/iotest"

	"github.com/envelope-zero/backend/pkg/importer/parser/ynab4"
	"github.com/stretchr/testify/assert"
)

func TestParseNoFile(t *testing.T) {
	_, err := ynab4.Parse(iotest.ErrReader(errors.New("Some reading error")))
	assert.NotNil(t, err, "Expected file opening to fail")
	assert.Contains(t, err.Error(), "could not read data from file", "Wrong error on parsing broken file: %s", err)
}

func TestParse(t *testing.T) {
	f, err := os.OpenFile("../../../../testdata/Budget.yfull", os.O_RDONLY, 0o400)
	if err != nil {
		assert.FailNow(t, "Failed to open the test file", err)
	}

	_, err = ynab4.Parse(f)
	assert.Nil(t, err, "Parsing failed", err)
}

func TestParseBrokenFile(t *testing.T) {
	f, err := os.OpenFile("../../../../testdata/EmptyFile.yfull", os.O_RDONLY, 0o400)
	if err != nil {
		assert.FailNow(t, "Failed to open the test file", err)
	}

	_, err = ynab4.Parse(f)
	assert.NotNil(t, err, "Expected parsing to fail")
	assert.Contains(t, err.Error(), "not a valid YNAB4 Budget.yfull file", "Wrong error on parsing broken file: %s", err)
}
