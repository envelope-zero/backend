# Upgrading

If upgrades between versions require manual actions, those are described here.

## v0.31.0 to v0.31.1

[v0.31.1](https://github.com/envelope-zero/backend/releases/tag/v0.31.1), fixes buggy behaviour from `v0.31.0` and simplifies base URL configuration at the same time.

To migrate, do the following:

- Set `API_URL` to the full path to the API, e.g. `https://ez.example.com/api`
- Upgrade to `v0.31.1`
- After the migration, you can remove `API_HOST` and `API_PATH` from your configuration

## v0.30.3 to v0.31.0

With [v0.31.0](https://github.com/envelope-zero/backend/releases/tag/v0.31.0), the host name and prefix are not auto-detected anymore. This was done with the `x-forwarded-host` and `x-forwarded-prefix` headers or the reuqest URL itself until now.

From `v0.31.0` on, this is done with the environment variables `API_HOST` and `API_PATH`.

To migrate, do the following:

- Set `API_HOST` to the scheme, hostname and port of your instance, e.g. `https://ez.example.com` or `http://localhost:8080`
- Set `API_PATH` to the prefix at which the API is available, e.g. `/api`.
- Upgrade to `v0.31.0`
- You can now unset the http headers `x-forwarded-host` and `x-forwarded-prefix` at your proxy if you want.

## v0.30.2 to v0.30.3

With [v0.30.3](https://github.com/envelope-zero/backend/releases/tag/v0.30.3), a database migration takes place. Both foreign key constraints and the requirement for transactions to have different source and destination accounts are now enforced.

Before upgrading, you need to ensure that:

- All resources reference other resources that are logically correct. For example, a transaction needs to use an account that belongs to the same budget as the transaction.
- Transactions have

If you have upgraded without taking those steps and are now encountering either `FOREIGN KEY constraint failed` or `CHECK constraint failed: source_destination_different` errors, you can roll back to version `v0.30.2` safely to perform the steps above.

The easiest way to perform those steps is to first upgrade the [frontend](https://github.com/envelope-zero/frontend) to version `0.17.0`. In the frontend, create envelopes for all transactions. Then, update your transactions to point to the correct accounts and envelopes. Upgrade the backend to `v0.30.3` afterwards.
