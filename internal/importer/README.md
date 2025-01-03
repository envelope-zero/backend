# importer

There are two types of importers:

- Budget importers. These import whole budgets at once.
- Transaction importers. These import transactions for a specified account. This is a two-step process: Transaction importers return a slice of `TransactionPreview` objects. These are returned by the API to allow users to edit the transactions before finally importing them.
