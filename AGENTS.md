# mbop

## Project Overview

mbop (Mock BOP) is a Go HTTP service that replaces the [BackOffice Proxy (BOP)][bop] in ephemeral
and security-compliance environments. It starts an HTTP server on port 8090 and proxies user, authentication,
and entitlement requests to configurable backends (Keycloak, AMS/OCM, or a built-in ephemeral
handler). It is containerized and deployed to OpenShift via Konflux/Tekton pipelines. The container
image is published to `quay.io/cloudservices/mbop`.

## Dependencies

- **Runtime:** Go 1.25.0, [go-chi/chi][chi] v5 (HTTP router), [pgx][pgx] v5 (PostgreSQL driver),
  [golang-migrate][golang-migrate] (schema migrations), [golang-jwt][golang-jwt] v5,
  [aws-sdk-go-v2][aws-sdk] (SES email), [ocm-sdk-go][ocm-sdk] (AMS/OCM user lookups),
  [platform-go-middlewares][platform-middlewares] (x-rh-identity header), [zap][zap] (structured
  logging via logr adapter)
- **Test:** [testify][testify] (Go unit tests), Mocha + Chai + chai-http (Node.js E2E tests in
  `test/`)
- **Lint:** [golangci-lint][golangci-lint] v2.11

## Development Commands

See [Development][readme-dev] in the README for the full command reference.

Key commands:

```sh
make build          # Build the binary from cmd/mbop/mbop.go
make test           # Run Go unit tests (go test ./...)
make lint           # Run golangci-lint with the project's linter set
make fix            # Run golangci-lint with --fix=true
```

CI runs `golangci-lint` via GitHub Actions (`lint.yml`) and Go unit tests with a PostgreSQL service
container plus E2E tests via docker compose (`package.yml`). The CI lint action uses golangci-lint
defaults with `only-new-issues: true`, which differs from the Makefile's explicit linter list.

## Architecture

mbop follows a standard Go project layout: `cmd/mbop/` (entrypoint), `internal/handlers/`
(stateless HTTP handlers), `internal/service/` (backend integrations), `internal/store/`
(persistence), `internal/config/` (environment-based singleton config), and `internal/models/`
(data types). The central pattern is module-switching via `USERS_MODULE`, `MAILER_MODULE`, and
`JWT_MODULE` environment variables -- each handler dispatches to the appropriate service
implementation at runtime through switch statements.

See [ARCHITECTURE.md][architecture] for detailed design decisions, dependency points, service layer
internals, database migration history, and key tradeoffs.

## Code Style

- **Linter:** golangci-lint with linters configured inline in the Makefile: `errcheck`, `gocritic`,
  `gofmt`, `goimports`, `gosec`, `gosimple`, `govet`, `ineffassign`, `revive`, `staticcheck`,
  `typecheck`, `unused`, `bodyclose`. No `.golangci.yml` config file exists.
- **Formatting:** `gofmt` and `goimports` (enforced by the linter set).
- **Go version:** 1.25.0 (specified in `go.mod`).
- **CI vs local:** The GitHub Actions lint workflow uses `golangci-lint-action@v9` with
  `only-new-issues: true` and golangci-lint's default linter set, which does not match the
  Makefile's explicit list. The Makefile configuration is the intended project standard.

## Common Mistakes

1. **Assuming a single user lookup path.** Handlers dispatch to different backends (`ams`, `mock`,
   `keycloak`, or catchall) via `switch config.Get().UsersModule`. When modifying user-related
   handlers, all code paths in the switch must be updated -- missing a branch causes silent
   fallthrough to the catchall handler.

2. **Confusing the two Keycloak integrations.** `service/keycloak/` is a token client that
   authenticates against Keycloak's OAuth endpoint. `service/keycloak-user-service/` is a separate
   HTTP client that talks to the Keycloak User Service API (a different service). They are not
   interchangeable. The token client provides Bearer tokens consumed by the user service client.

3. **Ignoring the in-memory store's limitations.** The in-memory store (`STORE_BACKEND=memory`)
   silently ignores `limit` and `offset` pagination parameters and returns all results. Tests
   that pass with the in-memory store may fail with PostgreSQL due to pagination, unique
   constraints, or transaction semantics.

4. **Overlooking the catchall's separate config path.** The `service/catchall/` package reads
   environment variables directly via `os.Getenv` instead of using `config.Get()`. Changes to
   the config package or its defaults do not affect the catchall handler.

5. **Mismatching CI lint configuration.** The Makefile `lint` target specifies an explicit list of
   linters, but the GitHub Actions workflow uses golangci-lint defaults. A change that passes
   `make lint` locally may fail in CI (or vice versa) if it triggers a linter that only one
   configuration enables.

## Testing

- **Unit tests:** `make test` runs `go test ./...`. CI runs these with a PostgreSQL service
  container.
- **E2E tests:** Node.js-based, located in `test/`. Require a running Keycloak + mbop environment
  via `docker compose -f deployments/compose.yaml up -d --build`. Run with
  `npm --prefix test test`.
- **Test data:** `test/data/` contains a Keycloak realm export (`redhat-external-realm.json`) and
  other fixtures used by both unit and E2E tests.

## Deployment

Container images are built using a multi-stage Dockerfile based on UBI9 (`go-toolset` builder,
`ubi9-minimal` runtime). The final image runs as non-root user `65532`.

Two image registries:

- `quay.io/cloudservices/mbop` -- pushed by `build_deploy.sh` (tags `latest` on master)
- `quay.io/redhat-user-workloads/hcc-fr-tenant/mbop/mbop` -- pushed by Tekton/Konflux pipelines

PR validation deploys to an ephemeral OpenShift environment using Bonfire and runs IQE smoke tests
(`pr_check.sh`).

[bop]: https://github.com/RedHatInsights/backoffice-proxy
[chi]: https://github.com/go-chi/chi
[pgx]: https://github.com/jackc/pgx
[golang-migrate]: https://github.com/golang-migrate/migrate
[golang-jwt]: https://github.com/golang-jwt/jwt
[aws-sdk]: https://github.com/aws/aws-sdk-go-v2
[ocm-sdk]: https://github.com/openshift-online/ocm-sdk-go
[platform-middlewares]: https://github.com/RedHatInsights/platform-go-middlewares
[zap]: https://github.com/uber-go/zap
[testify]: https://github.com/stretchr/testify
[golangci-lint]: https://golangci-lint.run/
[readme-dev]: ./README.md#development
[architecture]: ./ARCHITECTURE.md
