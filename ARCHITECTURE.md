# Architecture

Internal architecture of mbop (Mock BOP): design decisions, dependency points, and key tradeoffs.

## High-Level Structure

mbop is a Go HTTP service that replaces the [BackOffice Proxy][bop] in ephemeral and security-compliance
environments. It follows a standard Go project layout with a strict top-down dependency flow:
`cmd` -> `handlers` -> `service/*` + `store` -> `config` + `models`.

```
cmd/mbop/              Application entrypoint
internal/
  config/              Singleton environment-based configuration
  logger/              Global structured logger (zap via logr)
  middleware/          HTTP middleware (request logging)
  models/              Data transfer objects and domain types
  handlers/            HTTP handler functions (stateless, flat)
  service/
    catchall/          Ephemeral Keycloak-direct proxy (legacy path)
    keycloak/          Keycloak token client
    keycloak-user-service/  Keycloak User Service API client
    mailer/            Email sending (AWS SES or print-to-stdout)
    ocm/               OpenShift Cluster Manager (AMS) API client
  store/               Persistence layer (in-memory or PostgreSQL)
```

## Module-Switching Pattern

The central design pattern is **module-switching via environment variables**. Three configuration
knobs control which backend implementation is used for each capability:

| Variable        | Options                            | Effect                                       |
| --------------- | ---------------------------------- | -------------------------------------------- |
| `USERS_MODULE`  | `ams`, `mock`, `keycloak`, or `""` | Selects user lookup backend                  |
| `MAILER_MODULE` | `aws` or `print`                   | Selects email delivery backend               |
| `JWT_MODULE`    | `aws`, `keycloak`, or `""         `| Selects JWT/public-key retrieval backend      |

Every handler uses a `switch config.Get().UsersModule` (or the equivalent for mailer/JWT) to
dispatch to the appropriate service. When no module is set, requests fall through to the catchall
ephemeral handler.

This means the same binary serves different roles depending on environment variables. With no
modules set, everything uses the catchall Keycloak handler. With `ams`, it becomes a security-compliance BOP
replacement. With `keycloak`, it talks to the Keycloak User Service.

## Initialization Flow

Startup sequence in `cmd/mbop/mbop.go`:

1. **Config** -- `config.Get()` reads all environment variables on first call (lazy singleton).
2. **Logger** -- `logger.Init()` creates a zap development logger wrapped in `logr.Logger`.
3. **Store** -- `store.SetupStore()` reads `STORE_BACKEND` and initializes either the in-memory or
   PostgreSQL store. For Postgres, this includes connection setup, ping, and running all pending
   migrations.
4. **Router** -- Creates a `chi.Router` with two route groups: public routes (no auth) and
   identity-protected routes (behind `identity.EnforceIdentity` middleware).
5. **Mailer** -- `mailer.InitConfig()` pre-loads AWS SDK config if `MAILER_MODULE=aws`.
6. **Server** -- Launches HTTP on `PORT` (default 8090). If TLS certs exist at `CERT_DIR`, also
   launches HTTPS on `TLS_PORT` (default 8890). Blocks on `SIGINT`/`SIGTERM`.

## Router and Middleware

**Router:** [go-chi/chi][chi] v5.

**Middleware:**

- **Global:** `middleware.Logging` -- logs `RemoteAddr Method URL` for every request.
- **Group-level:** `identity.EnforceIdentity` (from [platform-go-middlewares][platform-middlewares])
  -- decodes and validates the `x-rh-identity` base64 header. Applied only to registration and
  allowlist routes.

**Handlers** are package-level functions in `internal/handlers/`, not methods on a struct. They are
stateless -- they read config via `config.Get()`, access the store via `store.GetStore()`, and
create service clients per-request.

### Route Table

| Method     | Path                               | Auth Required |
| ---------- | ---------------------------------- | ------------- |
| GET        | `/`                                | No            |
| GET/POST   | `/v*`, `/api/entitlements*`        | No            |
| GET        | `/v1/jwt`                          | No            |
| POST       | `/v1/users`                        | No            |
| POST       | `/v1/sendEmails`                   | No            |
| GET        | `/v3/accounts/{orgID}/users`       | No            |
| POST       | `/v3/accounts/{orgID}/usersBy`     | No            |
| GET        | `/v1/auth`                         | No            |
| GET/POST   | `/v1/registrations`                | x-rh-identity |
| DELETE     | `/v1/registrations/{uid}`          | x-rh-identity |
| GET        | `/v1/registrations/token`          | x-rh-identity |
| GET/POST/DELETE | `/api/mbop/v1/allowlist`      | x-rh-identity |

## Service Layer

### Keycloak Token Client (`service/keycloak/`)

Interface: `KeyCloak`. Authenticates against Keycloak's token endpoint to get admin access tokens.
Supports both `password` and `client_credentials` grant types. Used as a prerequisite by the
Keycloak User Service -- get a token here, then pass it to user service calls.

### Keycloak User Service (`service/keycloak-user-service/`)

Interface: `KeyCloakUserService`. Calls a separate Keycloak User Service API (not Keycloak itself)
configured via `KEYCLOAK_USER_SERVICE_*` env vars. This intermediary service provides a
higher-level user query API. Transforms `KeycloakResponses` into the common `models.Users` format.
Factory: `NewKeyCloakUserServiceClient()` only returns a client when `USERS_MODULE=keycloak`.

### OCM/AMS Service (`service/ocm/`)

Interface: `OCM`. Two implementations:

- `SDK` -- Real implementation using [ocm-sdk-go][ocm-sdk]. Authenticates via Cognito service
  account through OAuth, then queries the AMS Accounts Management API.
- `SDKMock` -- Returns synthetic user data for testing. Used when `USERS_MODULE=mock`.

Org admin status requires a **separate API call** (`GetOrgAdmin()`) because AMS separates account
data from RBAC. The handler fetches users first, then fetches role bindings, then merges the
results. The Keycloak User Service returns `is_org_admin` inline.

### Mailer (`service/mailer/`)

Interface: `Emailer`. Two implementations:

- `awsSESEmailer` -- Sends real emails via AWS SES v2.
- `printEmailer` -- Default. Prints email details to stdout.

`LookupEmailsForUsernames()` is a cross-service integration point -- the mailer resolves
username-only recipients to email addresses by calling either OCM, Keycloak, or mock depending on
`USERS_MODULE`.

### CatchAll / Ephemeral (`service/catchall/`)

`MBOPServer` is a self-contained handler that talks directly to Keycloak's admin REST API. This is
the original mbop behavior -- it reads all users from Keycloak's `redhat-external` realm. Created
once at package init in `handlers/catchall.go` and reused for all catchall requests. Can be
disabled via `DISABLE_CATCHALL=true`.

The catchall reads env vars directly (`os.Getenv`) rather than using the config singleton,
suggesting it predates the config package.

## Data Store

Interface: `Store` (composed of `RegistrationStore` and `AllowlistStore`).

**Accessor pattern:** A package-level function variable `GetStore func() Store` is set during
`SetupStore()`. This allows the implementation to be swapped at runtime (including for tests).

| Aspect      | In-Memory Store                   | PostgreSQL Store                          |
| ----------- | --------------------------------- | ----------------------------------------- |
| Backing     | Go slices                         | pgx driver                                |
| Pagination  | Ignores `limit`/`offset`          | Full SQL `LIMIT`/`OFFSET`                 |
| Uniqueness  | Manual loop checks                | PostgreSQL unique constraints (code 23505)|
| Persistence | None (lost on restart)            | Full persistence                          |

The in-memory store exists because mbop was originally ephemeral-only (no persistence needed). The
PostgreSQL store was added later for production use where registrations and allowlists need
persistence.

### Database Migrations

Managed by [golang-migrate][golang-migrate] with embedded filesystem source
(`//go:embed migrations`). Migrations run automatically on Postgres store setup via `m.Up()`.

| Migration | Change                                                                |
| --------- | --------------------------------------------------------------------- |
| 0         | Creates `registrations` table with UUID PK, JSONB extra, timestamps   |
| 1         | Adds unique index on `uid` alone                                      |
| 2         | Adds `display_name` column                                            |
| 3         | Adds `username` column                                                |
| 4         | Adds unique constraint on `(display_name, org_id)`                    |
| 5         | Creates `allowlist` table with composite PK `(ip_block, org_id)`      |

All migrations are embedded into the binary at compile time, so no external migration files are
needed at deployment.

## External Dependencies

| System                   | Service Package              | Auth Method                         | Purpose                                |
| ------------------------ | ---------------------------- | ----------------------------------- | -------------------------------------- |
| Keycloak (admin API)     | `service/catchall/`          | OAuth2 password grant               | Ephemeral mode: read realm users       |
| Keycloak (token endpoint)| `service/keycloak/`          | Password or client_credentials      | Get tokens for User Service calls      |
| Keycloak User Service    | `service/keycloak-user-service/` | Bearer token                    | Query users by username, org_id, email |
| AMS                      | `service/ocm/`               | Cognito service account OAuth       | Query user accounts and role bindings  |
| AWS Cognito              | `service/ocm/` (via OCM SDK) | Client credentials                  | Token acquisition for AMS API          |
| AWS SES v2               | `service/mailer/`            | Static AWS credentials              | Send emails                            |
| JWK endpoint             | `handlers/jwt_v1_handler.go` | None (public)                       | Fetch JWK keys, convert to PEM        |
| PostgreSQL               | `store/`                     | Username/password                   | Persist registrations and allowlists   |

## Key Tradeoffs

- **Module-switching vs. dependency injection.** Services are created inline per-request via switch
  statements rather than injected at startup. This keeps the code simple and avoids framework
  dependencies, but leads to duplicated dispatch logic across handlers and repeated connection
  setup overhead.
- **Global state.** Config, store, logger, and the catchall server are all package-level singletons
  or function variables. This simplifies handler signatures but makes testing require explicit
  reset calls (`config.Reset()`, reassigning `store.GetStore`).
- **In-memory store fallback.** Allows zero-dependency local development but silently drops data
  on restart and ignores pagination parameters, which can mask bugs in query logic.

[bop]: https://github.com/RedHatInsights/backoffice-proxy
[chi]: https://github.com/go-chi/chi
[platform-middlewares]: https://github.com/RedHatInsights/platform-go-middlewares
[ocm-sdk]: https://github.com/openshift-online/ocm-sdk-go
[golang-migrate]: https://github.com/golang-migrate/migrate
