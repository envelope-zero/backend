.PHONY: setup-pre-commit-ci
setup-pre-commit-ci:
# renovate: datasource=github-releases depName=golangci/golangci-lint
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.1
# renovate: datasource=github-releases depName=swaggo/swag
	go install github.com/swaggo/swag/cmd/swag@v1.16.2

.PHONY: setup
setup: setup-pre-commit-ci
	pre-commit install --hook-type commit-msg --hook-type pre-commit
	go install github.com/cosmtrek/air@latest

.PHONY: devserver
devserver:
	GIN_MODE=debug air

.PHONY: test
test:
	go test ./... -covermode=atomic -coverprofile=coverage.out -count=1

.PHONY: coverage
coverage: test
	go tool cover -html=coverage.out

VERSION ?= $(shell git rev-parse HEAD)
.PHONY: build
build:
	go build -ldflags "-X github.com/envelope-zero/backend/v3/pkg/router.version=${VERSION}"


.PHONY: docs
docs:
	swag init --parseDependency --output ./api
