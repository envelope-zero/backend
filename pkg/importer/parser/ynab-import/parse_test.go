package ynabimport

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/stretchr/testify/assert"
)

// TestParse verifies that parsing is correct for valid files.
func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		length int
	}{
		{"Empty file", "empty.csv", 0},
		{"With content", "comdirect-ynap.csv", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.OpenFile(fmt.Sprintf("../../../../testdata/importer/ynab-import/%s", tt.file), os.O_RDONLY, 0o400)
			if err != nil {
				assert.FailNow(t, "Failed to open the test file", err)
			}

			transactions, err := Parse(f, models.Account{})
			assert.Nil(t, err, "Parsing failed")
			assert.Len(t, transactions, tt.length, "Wrong number of transactions has been parsed")

			for _, transaction := range transactions {
				assert.True(t, transaction.Transaction.Amount.IsPositive(), "Transaction amount is not positive: %s", transaction.Transaction.Amount)
			}
		})
	}
}

// TestReadError verifies that the csvReadError helper method returns the correct result.
func TestReadError(t *testing.T) {
	f, err := os.OpenFile(fmt.Sprintf("../../../../testdata/importer/ynab-import/%s", "comdirect-ynap.csv"), os.O_RDONLY, 0o400)
	if err != nil {
		assert.FailNow(t, "Failed to open the test file", err)
	}

	reader := csv.NewReader(f)
	reader.Read()

	_, err = csvReadError(reader, errors.New("Test error"))
	assert.Equal(t, "error in line 1 of the CSV: Test error", err.Error(), "Generated error message is wrong")
}

// TestErrors tests the various error conditions.
func TestErrors(t *testing.T) {
	tests := []struct {
		file    string
		message string
	}{
		{"error-date.csv", "error in line 4 of the CSV: could not parse time: parsing time"},
		{"error-decimal-inflow.csv", "error in line 4 of the CSV: inflow could not be parsed to a decimal"},
		{"error-decimal-outflow.csv", "error in line 2 of the CSV: outflow could not be parsed to a decimal"},
		{"error-missing-amount.csv", "error in line 3 of the CSV: no amount is set for the transaction"},
		{"error-amount-zero.csv", "error in line 4 of the CSV: the amount for a transaction must not be 0"},
		{"error-outflow-and-inflow.csv", "error in line 2 of the CSV: both outflow and inflow are set for the transaction"},
	}

	for _, tt := range tests {
		f, err := os.OpenFile(fmt.Sprintf("../../../../testdata/importer/ynab-import/%s", tt.file), os.O_RDONLY, 0o400)
		if err != nil {
			assert.FailNow(t, "Failed to open the test file", err)
		}

		_, err = Parse(f, models.Account{})
		if assert.NotNil(t, err, "No parsing error where an error is expected for file %s", tt.file) {
			assert.Contains(t, err.Error(), tt.message, "Error message for file %s does not contain expected content", tt.file)
		}
	}
}
