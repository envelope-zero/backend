# Upgrading

:warning: You cannot skip major versions on upgrades. You have to upgrade to the latest release of each major version before updating to the next major version.

If upgrades between versions require manual actions, those are described here.

# to v4.0.0

1. Upgrade to v3.22.2 before upgrading to v4.0.0
2. Upgrade to v4.0.0

For breaking changes, see the release notes

# to v3.0.0

v3.0.0 does not require manual steps. With v3.0.0, account names must be unique per budget.

# to v2.0.0

v2.0.0 does not require manual steps. With v2.0.0, spent amounts are now negative values instead of positive values.

## < v1.0.0 to v1.0.0

If you are running a version below `v0.35.0`, you _must_ upgrade to `v1.0.0` before you upgrade any further.

## v0.32.0 to v0.33.0

[v0.33.0](https://github.com/envelope-zero/backend/releases/tag/v0.33.0) removes support for postgresql.

If you are currently using postgresql, you need to:

1. Export your data with `pg_dump --data-only`.
2. Upgrade to `v0.33.0` and then re-import the data with `.read database-dump.sql` to the sqlite database.

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
