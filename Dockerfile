FROM golang:1.19.1-alpine as builder

RUN apk add --no-cache make

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=0.0.0
RUN CGO_ENABLED=0 make build VERSION=${VERSION}

# Build final image
FROM scratch
WORKDIR /
COPY --from=builder /app/backend /backend
ENTRYPOINT ["/backend"]
