FROM golang:1.18-alpine as builder

RUN apk add --no-cache make

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY internal ./internal
COPY main.go Makefile ./

ARG VERSION=0.0.0
RUN CGO_ENABLED=0 make build VERSION=${VERSION}

# Build final image
FROM scratch
WORKDIR /
COPY --from=builder /app/backend /backend
ENTRYPOINT ["/backend"]
