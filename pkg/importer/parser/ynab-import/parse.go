package ynabimport

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/importer"
	"github.com/envelope-zero/backend/v3/pkg/importer/helpers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/shopspring/decimal"
)

// This function parses the YNAB import CSV files.
func Parse(f io.Reader, account models.Account) ([]importer.TransactionPreview, error) {
	reader := csv.NewReader(f)

	// We can reuse the array in the background to improve performance
	reader.ReuseRecord = true

	var transactions []importer.TransactionPreview

	// Skip the first line
	_, err := reader.Read()
	if err == io.EOF {
		return []importer.TransactionPreview{}, nil
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return csvReadError(reader, fmt.Errorf("could not read line in CSV: %w", err))
		}

		date, err := time.Parse("01/02/2006", record[Date])
		if err != nil {
			return csvReadError(reader, fmt.Errorf("could not parse time: %w", err))
		}

		t := importer.TransactionPreview{
			Transaction: models.TransactionCreate{
				Date:          date,
				AvailableFrom: types.NewMonth(date.Year(), date.Month()),
				ImportHash:    helpers.Sha256String(strings.Join(record, ",")),
				Note:          record[Memo],
				BudgetID:      account.BudgetID,
			},
		}

		// Set the source and destination account
		if record[Outflow] != "" && record[Inflow] != "" {
			return csvReadError(reader, errors.New("both outflow and inflow are set for the transaction"))
		} else if record[Outflow] == "" && record[Inflow] == "" {
			return csvReadError(reader, errors.New("no amount is set for the transaction"))
		} else if record[Outflow] != "" {
			t.Transaction.SourceAccountID = account.DefaultModel.ID
			t.DestinationAccountName = record[Payee]

			amount, err := decimal.NewFromString(record[Outflow])
			if err != nil {
				return csvReadError(reader, errors.New("outflow could not be parsed to a decimal"))
			}

			t.Transaction.Amount = amount
		} else {
			t.Transaction.DestinationAccountID = account.DefaultModel.ID
			t.SourceAccountName = record[Payee]

			amount, err := decimal.NewFromString(record[Inflow])
			if err != nil {
				return csvReadError(reader, errors.New("inflow could not be parsed to a decimal"))
			}

			t.Transaction.Amount = amount
		}

		if t.Transaction.Amount.IsZero() {
			return csvReadError(reader, errors.New("the amount for a transaction must not be 0"))
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
