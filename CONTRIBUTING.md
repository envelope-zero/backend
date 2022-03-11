# Contributing

Contributions are welcome. Please note the [Code of Conduct](CODE_OF_CONDUCT.md) and set up pre-commit as described below.

## Tool & Repository setup

### Recommendations

CI pipelines check all code changes when you open a PR. To not have to go back and fix all issues manually, the following setup is recommended.

You will need the following tools:

- [go](https://go.dev/). For the specific version used, check the [pre-commit workflow](.github/workflows/pre-commit.yml) at the `go-version` configuration
- [pre-commit](https://pre-commit.com/)

Once this is done, run the following:

```sh
# Linters used with pre-commit
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Set up pre-commit hooks
pre-commit install --hook-type commit-msg --hook-type pre-commit
```

### Hot reload

If you want to hot reload the server, using `air` is recommended. Get it with

```sh
go install github.com/cosmtrek/air@latest
```

You can then run `air` in the repository root, which will build and rebuild the project every time the code changes.

## Commit messages

This project uses [Conventional commits](https://www.conventionalcommits.org/en/v1.0.0-beta.4/)
to enable better overview over changes and enables automated tooling based on commit messages.
