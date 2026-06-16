# Mock BOP (mbop)

A mock replacement for the [BackOffice Proxy (BOP)][bop] service, used in ephemeral and security-compliance
environments. mbop starts an HTTP server on port 8090 and proxies requests to different backends
(Keycloak, AMS/OCM, or a built-in ephemeral handler) depending on configuration.

It is designed to run alongside a [Keycloak][keycloak] instance, where it can fetch and return
information about users and realms in the `redhat-external` realm.

For internal design decisions and architecture details, see [ARCHITECTURE.md][architecture].

## Prerequisites

- **Go 1.25.0+**
- **Keycloak** -- A running Keycloak server with a pre-configured realm named `redhat-external`.
  A [realm template][realm-template] is provided that defines the realm and a test user. Import it
  to get started.

The `redhat-external` realm must have users with the following custom attributes:

- `is_active` (Boolean)
- `is_org_admin` (Boolean)
- `is_internal` (Boolean)
- `account_id` (String)
- `org_id` (String)
- `entitlements` (String)
- `account_number` (String)

## Supported API Paths

| Method   | Path                            | Description                                              |
| -------- | ------------------------------- | -------------------------------------------------------- |
| GET      | `/`                             | Status check endpoint                                    |
| POST     | `/v1/users`                     | Fetch Keycloak users                                     |
| GET      | `/v1/jwt`                       | Returns the `redhat-external` realm public key           |
| GET      | `/v1/auth`                      | Basic auth login; returns a token and user entity         |
| GET/POST | `/v1/accounts`                  | Query users for a specific account                       |
| GET      | `/v2/accounts`                  | Query users with filter query parameters                 |
| GET      | `/v3/accounts/{orgID}/users`    | Query users by org ID (v3)                               |
| POST     | `/v3/accounts/{orgID}/usersBy`  | Query users by org ID with body filters (v3)             |
| POST     | `/v1/sendEmails`                | Send emails via configured mailer backend                |
| *        | `/api/entitlements/v1/services` | Returns user entitlements from the Identity header       |
| GET/POST | `/v1/registrations`             | List or create satellite registrations (requires identity)|
| DELETE   | `/v1/registrations/{uid}`       | Delete a registration (requires identity)                |
| GET      | `/v1/registrations/token`       | Generate a registration token (requires identity)        |
| *        | `/api/mbop/v1/allowlist`        | Manage IP allowlist entries (requires identity)          |

Routes marked "requires identity" expect an `x-rh-identity` base64-encoded header.

## Running

### Environment Variables

mbop is configured entirely through environment variables. The three module variables control which
backend is used for each capability:

| Variable        | Default | Options                            | Purpose                     |
| --------------- | ------- | ---------------------------------- | --------------------------- |
| `USERS_MODULE`  | (empty) | `ams`, `mock`, `keycloak`, or `""` | User lookup backend         |
| `JWT_MODULE`    | (empty) | `aws`, `keycloak`, or `""`         | JWT/public-key backend      |
| `MAILER_MODULE` | `print` | `aws`, `print`                     | Email delivery backend      |
| `STORE_BACKEND` | `memory`| `memory`, `postgres`               | Persistence backend         |

Additional variables for Keycloak, database, AWS SES, and AMS/Cognito are documented in
`internal/config/config.go`.

### Run Locally with Go

Start a Keycloak server that imports the demo realm:

```sh
podman run -it --name keycloak -p 8080:8080 \
    -e KEYCLOAK_ADMIN_USER=admin \
    -e KEYCLOAK_ADMIN_PASSWORD=change_me \
    -e KEYCLOAK_IMPORT=/opt/keycloak/data/import/redhat-external-realm.json \
    -v "${PWD}/test/data/redhat-external-realm.json:/opt/keycloak/data/import/redhat-external-realm.json:z" \
    quay.io/keycloak/keycloak:15.0.2
```

Then build and run mbop:

```sh
make build

KEYCLOAK_SERVER='http://localhost:8080' \
KEYCLOAK_USERNAME='admin' \
KEYCLOAK_PASSWORD='change_me' \
./mbop
```

Or use `make run` to build and start in one step.

### Run with Docker Compose

A compose file is provided that starts both mbop and Keycloak together:

```sh
docker compose -f deployments/compose.yaml up -d --build
```

If using podman-compose with SELinux enforced, set `SELINUX_LABEL=:z` before running:

```sh
cp deployments/podman-compose-env deployments/.env
podman-compose -f deployments/compose.yaml up -d --build
```

### Run with a Container

```sh
podman build -t localhost/mbop:dev .
podman run -it --rm --name mbop -p 8090:8090 \
    -e KEYCLOAK_SERVER='http://localhost:8080' \
    -e KEYCLOAK_USERNAME='admin' \
    -e KEYCLOAK_PASSWORD='change_me' \
    localhost/mbop:dev
```

## Development

### Build

```sh
make build
```

Produces the `mbop` binary from `cmd/mbop/mbop.go`.

### Test

Unit tests:

```sh
make test
```

E2E tests (requires a running Keycloak + mbop environment):

```sh
docker compose -f deployments/compose.yaml up -d --build
npm --prefix test test
```

### Lint

```sh
make lint
```

Runs [golangci-lint][golangci-lint] with the following linters: `errcheck`, `gocritic`, `gofmt`,
`goimports`, `gosec`, `gosimple`, `govet`, `ineffassign`, `revive`, `staticcheck`, `typecheck`,
`unused`, `bodyclose`.

To auto-fix linter issues:

```sh
make fix
```

## Deployment

Container images are built using a multi-stage [Dockerfile][dockerfile] based on UBI9. The builder
stage uses `registry.access.redhat.com/ubi9/go-toolset` and the runtime stage uses
`registry.access.redhat.com/ubi9-minimal`. The final image runs as non-root user `65532`.

CI pushes images to `quay.io/cloudservices/mbop` (via `build_deploy.sh`) and
`quay.io/redhat-user-workloads/hcc-fr-tenant/mbop/mbop` (via Tekton/Konflux pipelines).

## License

No license file is currently present in this repository.

[bop]: https://github.com/RedHatInsights/backoffice-proxy
[keycloak]: https://www.keycloak.org
[architecture]: ./ARCHITECTURE.md
[realm-template]: ./test/data/redhat-external-realm.json
[golangci-lint]: https://golangci-lint.run/
[dockerfile]: ./Dockerfile
