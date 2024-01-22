package ynabimport

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/envelope-zero/backend/v5/pkg/importer"
	"github.com/envelope-zero/backend/v5/pkg/importer/helpers"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/shopspring/decimal"
)

// This function parses the YNAB import CSV files.
func Parse(f io.Reader, account models.Account) ([]importer.TransactionPreview, error) {
	reader := csv.NewReader(f)

	// We can reuse the array in the background to improve performance
	reader.ReuseRecord = true

	var transactions []importer.TransactionPreview

	// First line contains headers
	headerRow, err := reader.Read()
	if err == io.EOF {
		return []importer.TransactionPreview{}, nil
	} else if err != nil {
		// csv reading always returns usable error messages
		return []importer.TransactionPreview{}, err
	}

	// Build map for header keys
	headers := map[string]int{}
	for i := range headerRow {
		headers[headerRow[i]] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// csv reading always returns usable error messages
			return []importer.TransactionPreview{}, err
		}

		date, err := time.Parse("01/02/2006", record[headers["Date"]])
		if err != nil {
			return csvReadError(reader, fmt.Errorf("could not parse time: %w", err))
		}

		t := importer.TransactionPreview{
			Transaction: models.Transaction{
				Date:          date,
				AvailableFrom: types.NewMonth(date.Year(), date.Month()),
				ImportHash:    helpers.Sha256String(strings.Join(record, ",")),
				Note:          record[headers["Memo"]],
			},
		}

		// Set the source and destination account
		if record[headers["Outflow"]] != "" && record[headers["Inflow"]] != "" {
			return csvReadError(reader, errors.New("both outflow and inflow are set for the transaction"))
		} else if record[headers["Outflow"]] == "" && record[headers["Inflow"]] == "" {
			return csvReadError(reader, errors.New("no amount is set for the transaction"))
		} else if record[headers["Outflow"]] != "" {
			t.Transaction.SourceAccountID = account.DefaultModel.ID
			t.DestinationAccountName = record[headers["Payee"]]

			amount, err := decimal.NewFromString(record[headers["Outflow"]])
			if err != nil {
				return csvReadError(reader, errors.New("outflow could not be parsed to a decimal"))
			}

			t.Transaction.Amount = amount
		} else {
			t.Transaction.DestinationAccountID = account.DefaultModel.ID
			t.SourceAccountName = record[headers["Payee"]]

			amount, err := decimal.NewFromString(record[headers["Inflow"]])
			if err != nil {
				return csvReadError(reader, errors.New("inflow could not be parsed to a decimal"))
			}

			t.Transaction.Amount = amount
		}

		// Ignore transactions that have an amount of 0
		if t.Transaction.Amount.IsZero() {
			continue
		}

		transactions = append(transactions, t)
	}

	return transactions, nil
}

// csvReadError returns the an error with the format string, including the line of the input
// the error occurred in in the message.
func csvReadError(r *csv.Reader, err error) ([]importer.TransactionPreview, error) {
	// always use the first field, we are only interested in the line
	line, _ := r.FieldPos(1)

	return []importer.TransactionPreview{}, fmt.Errorf("error in line %d of the CSV: %w", line, err)
}
