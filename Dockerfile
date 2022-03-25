# syntax=docker/dockerfile:1
FROM golang:1.18-alpine as builder

# needed for github.com/mattn/go-sqlite3
RUN apk add --no-cache gcc g++

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY internal ./internal
COPY main.go ./
RUN go build -o /backend

# Build final image
FROM scratch
WORKDIR /
COPY --from=builder /backend /backend
ENTRYPOINT ["/backend"]
