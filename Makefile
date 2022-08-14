.PHONY: setup
setup:
	pre-commit install --hook-type commit-msg --hook-type pre-commit

.PHONY: devserver
devserver:
	GIN_MODE=debug air

.PHONY: test
test:
	go test ./... -covermode=count -coverprofile=coverage.out -count=1

.PHONY: coverage
coverage: test
	go tool cover -html=coverage.out

VERSION ?= $(shell git rev-parse HEAD)
.PHONY: build
build:
	go build -ldflags "-X github.com/envelope-zero/backend/internal/router.version=${VERSION}"
