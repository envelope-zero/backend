# Contributing

Contributions are welcome. Please note the [Code of Conduct](CODE_OF_CONDUCT.md) and set up pre-commit as described below.

## Tool & Repository setup

You will need the following tools:

- [go](https://go.dev/). For the specific version used, check the [pre-commit workflow](.github/workflows/pre-commit.yml) at the `go-version` configuration
- [pre-commit](https://pre-commit.com/)

Once those are installed, run `make setup` to perform the repository setup.

Ensure that you have `~/go/bin` in your `PATH` environment variable since the go tooling will be installed there.

## Development commands

- `make devserver` will start a development server on port 8080 and and rebuild the project every time the code changes.
- `make test` runs all tests
- `make coverage` runs all tests and opens the coverage report in your browser
- `make build` builds the software with production configuration

## Comments

- Use `// FIXME: What to fix` or `// BUG: what is broken` to make pre-commit fail as a reminder to fix something before opening a PR
- Use `// TODO: What needs to be done` for things that need to be changed at a later point in time

## API documentation

API documentation is auto-generated with swagger. All struct fields used in the API must have

- the `json` field
- the `example` field, where a swagger-parseable example can be set (go basic data types)
- A comment indicating the use of the field

## Commit messages

This project uses [Conventional commits](https://www.conventionalcommits.org/en/v1.0.0-beta.4/)
to enable better overview over changes and enables automated tooling based on commit messages.

## Tests & test coverage

We try to test as much as possible. However, tests are only one indicator for a functioning codebase.
We do not aim to cover 100% of code, but use test coverage as a helpful indicator to point out code paths we should test.

Please do:

- try to add tests for everything you add to the codebase. If you're unsure about how to test, please open a pull request and ask for input so we can work on it together!
- add regression tests for bug fixes

## Releases & Versioning

Releases are fully automated and happen on every _feature_ and _bug fix_ that is merged into the `main` branch.

Maintainers can manually trigger the release of a new version by creating the corresponding tag and pushing it.
This is currently not used as releases are only needed on new features and bug fixes.

Versioning strictly adheres to [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html) with the following additions:

- If a release with only dependency updates is made, it incerases the `PATCH` version.

## Common errors

### pre-commit fails in GitHub action

If pre-commit runs successfully on your local machine, but errors in the GitHub action, it's likely that the `swaggo/swag/cmd` module has been updated, but you still have the old version locally.

Run `make setup` again to update and then `pre-commit run --all-files` to fix your commit.
