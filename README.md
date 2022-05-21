# Envelope Zero backend

[![Release](https://img.shields.io/github/release/envelope-zero/backend.svg?style=flat-square)](https://github.com/envelope-zero/backend/releases/latest) [![Go Reference](https://pkg.go.dev/badge/github.com/envelope-zero/backend.svg)](https://pkg.go.dev/github.com/envelope-zero/backend) [![Go Report Card](https://goreportcard.com/badge/github.com/envelope-zero/backend)](https://goreportcard.com/report/github.com/envelope-zero/backend)

Envelope Zero is fundamentally rooted in two ideas:

- Using the [envelope method](https://en.wikipedia.org/wiki/Envelope_system) to budget expenses into envelopes.
- Zero Based Budeting, meaning that you assign all your money to an envelope. Saving for a vacation? Create an envelope and archive it after your vacation. Rent? Create an envelope that gets a fixed amount of money added every month.

## Features

For a high level view of planned complex features, check the [milestones](https://github.com/envelope-zero/backend/milestones).

To see all planned features, check the [list of issues with the **enhancement** label](https://github.com/envelope-zero/backend/labels/enhancement).

## Usage

The recommended and only supported way for production deployments is to run the backend with [the OCI image](https://github.com/envelope-zero/backend/pkgs/container/backend).

### Configuration

:warning: If you do not configure a postgresql database, sqlite will automatically be used. Mount a persistent volume to the `/data` directory - this is where the sqlite database is stored. If you do not do this, you will lose all data every time the container is deleted.

If you serve the backend with a reverse proxy under a different path than `/`, note the following:

- Your reverse proxy must set the `x-forwarded-host` header so that correct URIs are generated
- If you do not serve the backend at the prefix `/api` (required by the [frontend](https://github.com/envelope-zero/frontend)), your reverse proxy needs to set the `x-forwarded-prefix` header

The backend can be configured with the following environment variables. None are required.

| Name                 | Type                      | Default                                              | Description                                                                                                                                                       |
| -------------------- | ------------------------- | ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GIN_MODE`           | One of `release`, `debug` | `release`                                            | The mode that gin runs in. Only set this to `debug` on your development environment!                                                                              |
| `PORT`               | `number`                  | `8080`                                               | The port the backend listens on                                                                                                                                   |
| `LOG_FORMAT`         | One of `json`, `human`    | `json` if `GIN_MODE` is `release`, otherwise `human` | If log output is written human readable or as JSON.                                                                                                               |
| `DB_HOST`            | `string`                  |                                                      | hostname or address of postgresql                                                                                                                                 |
| `DB_USER`            | `string`                  |                                                      | username for the postgresql connection                                                                                                                            |
| `DB_PASSWORD`        | `string`                  |                                                      | password for `DB_USER`                                                                                                                                            |
| `DB_NAME`            | `string`                  |                                                      | name of the database to use                                                                                                                                       |
| `CORS_ALLOW_ORIGINS` | `string`                  |                                                      | hosts that are allowed to use cross origin requests, separated by spaces. Only set this when your frontend runs on a different host and/or port than the backend! |

### Deployment methods

If you want to deploy with a method not listed here, you are welcome to open a discussion to ask any questions needed so that this documentation can be improved.

#### On Kubernetes

You can run the backend on any Kubernetes cluster with a supported version using the [morremeyer/generic]() helm chart with the following values:

```yaml
image:
  repository: ghcr.io/envelope-zero/backend
  tag: v0.2.1

# Only set this when you want to use sqlite as database backend.
# In this case, you need to make sure the database is regularly backed up!
persistence:
  enabled: true
  mountPath: /app/data

ports:
  - name: http
    containerPort: 8080
    protocol: TCP

ingress:
  enabled: true
  hosts:
    - host: envelope-zero.example.com
      paths:
        - path: /api
  tls:
    - hosts:
        - envelope-zero.example.com
```

## Supported Versions

This project is under heavy development. Therefore, only the latest release is supported.

Please check the [releases page](https://github.com/envelope-zero/backend/releases) for the latest release.

## Contributing

Please see [the contribution guidelines](CONTRIBUTING.md).
