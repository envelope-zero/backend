FROM golang:1.19.4-alpine as builder

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

# Keep "maintainer" and "org.opencontainers.image.authors" in sync
LABEL "maintainer"="Envelope Zero Maintainers <envelope-zero@maurice-meyer.de>"
LABEL "org.opencontainers.image.authors"="Envelope Zero Maintainers <envelope-zero@maurice-meyer.de>"
LABEL "org.opencontainers.image.description"="Backend for Envelope Zero"
LABEL "org.opencontainers.image.documentation"="https://github.com/envelope-zero/backend"
LABEL "org.opencontainers.image.licenses"="AGPL-3.0-or-later"
LABEL "org.opencontainers.image.source"="https://github.com/envelope-zero/backend"
LABEL "org.opencontainers.image.url"="https://github.com/envelope-zero/backend"
LABEL "org.opencontainers.image.vendor"="Envelope Zero Maintainers"
