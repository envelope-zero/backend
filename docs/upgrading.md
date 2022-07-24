# Upgrading

If upgrades between versions require manual actions, those are described here

## v0.30.2 to v0.30.3

With [v0.30.3](https://github.com/envelope-zero/backend/releases/tag/v0.30.3), a database migration takes place. Both foreign key constraints and the requirement for transactions to have different source and destination accounts are now enforced.

Before upgrading, you need to ensure that:

- All resources reference other resources that are logically correct. For example, a transaction needs to use an account that belongs to the same budget as the transaction.
- Transactions have

If you have upgraded without taking those steps and are now encountering either `FOREIGN KEY constraint failed` or `CHECK constraint failed: source_destination_different` errors, you can roll back to version `v0.30.2` safely to perform the steps above.

The easiest way to perform those steps is to first upgrade the [frontend](https://github.com/envelope-zero/frontend) to version `0.17.0`. In the frontend, create envelopes for all transactions. Then, update your transactions to point to the correct accounts and envelopes. Upgrade the backend to `v0.30.3` afterwards.
