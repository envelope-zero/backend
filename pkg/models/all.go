package models

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// TransactionSums returns the sum of all transactions matching two Transaction structs
//
// The incoming Transactions fields is used to add the amount of all matching transactions to the overall sum
// The outgoing Transactions fields is used to subtract the amount of all matching transactions from the overall sum.
func TransactionsSum(incoming, outgoing Transaction) decimal.Decimal {
	var outgoingSum, incomingSum decimal.NullDecimal

	_ = DB.Table("transactions").
		Where(&outgoing).
		Select("SUM(amount)").
		Row().
		Scan(&outgoingSum)

	_ = DB.Table("transactions").
		Where(&incoming).
		Select("SUM(amount)").
		Row().
		Scan(&incomingSum)

	return incomingSum.Decimal.Sub(outgoingSum.Decimal)
}

// RawTransactions returns a list of transactions for a raw SQL query.
func RawTransactions(query string) ([]Transaction, error) {
	var transactions []Transaction

	err := DB.Raw(query).Scan(&transactions).Error
	if err != nil {
		return []Transaction{}, fmt.Errorf("getting transactions with query '%v' failed: %w", query, err)
	}

	return transactions, nil
}
