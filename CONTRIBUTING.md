# Contributing

Contributions are welcome. Please note the [Code of Conduct](CODE_OF_CONDUCT.md) and set up pre-commit as described below.

## Tool & Repository setup

You will need the following tools:

- [go](https://go.dev/). For the specific version used, check the [pre-commit workflow](.github/workflows/pre-commit.yml) at the `go-version` configuration
- [pre-commit](https://pre-commit.com/)

Once those are installed, run `make setup` to perform the repository setup.

## Development server

Run `make devserver` in the repository root, which will build and rebuild the project every time the code changes.

## Commit messages

This project uses [Conventional commits](https://www.conventionalcommits.org/en/v1.0.0-beta.4/)
to enable better overview over changes and enables automated tooling based on commit messages.

## Tests & test coverage

The test coverage goal is 100%. Please try to add tests for everything you add to the codebase. If in doubt, you’re always welcome to open an issue and ask for help.

To run tests, run `make test`. To show the test coverage graphically in your browser, run `make coverage`.
