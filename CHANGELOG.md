# Changelog

All notable changes to this project are documented here.

English | **[简体中文](CHANGELOG.zh-CN.md)**

Changes are listed by commit date in reverse chronological order using
`YYYY-MM-DD`, with Added, Changed, and Fixed categories within each day.

## 2026-10-06

### Changed

- Database configuration now uses separate host, port, user, password, and
  database fields, with PostgreSQL SSL mode. Server startup and manual migrations
  build connection strings internally; existing YAML must replace `db.dsn`.

## 2026-08-09

### Changed

- Built-in CORS policy that allows every origin without credentials, handles
  preflight requests, and exposes request IDs to browser clients.

## 2026-08-07

### Changed

- Fixed HTTP read, write, idle, header, and request-body safeguards.

## 2026-08-06

### Changed

- Deprecated operations retain OpenAPI metadata validation only; the runtime
  `Deprecation` / `Sunset` response-header middleware has been removed.
- Helm deployments use `terminationGracePeriodSeconds` for bounded graceful
  shutdown, without a separate readiness drain delay.

## 2026-07-20

### Fixed

- Outbound retries no longer wait after the final attempt or consume the final
  error response body before callers can inspect it.
- HTTP listener startup failures are returned deterministically instead of
  racing with graceful-shutdown cancellation and potentially exiting cleanly.

## 2026-07-19

Established the initial template baseline (formerly `0.1.0`) and hardened API
boundaries, deployment, and supply-chain safeguards.

### Added

- OpenAPI-first server stubs, strict Gin handlers, a Go client SDK, and an
  embedded runtime specification generated from `spec/openapi.yaml`.
- `/healthz`, `/readyz`, `/version`, and Prometheus `/metrics` endpoints.
- Optional Gorm database support for PostgreSQL, MySQL, and SQLite with
  connection pooling, startup ping checks, and OpenTelemetry SQL tracing.
- OpenTelemetry traces and metrics through OTLP HTTP plus Prometheus pull
  export, with structured slog correlation for trace, span, and request IDs.
- `pkg/httpx` client helpers with JSON operations, retries with exponential
  backoff and `Retry-After` support, W3C trace propagation, and per-attempt
  structured logging.
- Docker images, a Helm chart for the backend and frontend, health probes,
  optional HPA and Ingress resources, and Secret-backed server configuration.
- Vite + React + TypeScript frontend scaffolding with an independent nginx
  deployment.
- Project initialization tooling, Makefile workflows, GitHub Actions CI,
  Dependabot configuration, DCO checks, linting, and security-audit targets.
- Runtime request validation against the embedded OpenAPI contract.
- A consistent `api.Error` response for request parsing, routing, handler,
  response serialization, and panic failures.
- Typed client response fields for standard 400, 404, 405, 413, and 500 API
  errors.
- An extensible global Gin middleware chain with a documented custom-handler
  extension point.
- `make tools` for installing pinned versions of the development tools.
- URL-prefix API versioning policy for business endpoints, with unversioned
  operational probe exceptions documented in the OpenAPI contract.
- A pinned `oasdiff` compatibility check via `make contract-check`, enforced on
  pull requests against the target branch's OpenAPI contract.
- A single Go 1.26.5 toolchain across `go.mod`, local checks, CI, and the
  backend builder image.
- Explicitly versioned Docker base-image tags and immutable GitHub Actions
  references, plus `make supply-chain-check` to detect pin drift.

### Changed

- The default backend port is `8000`.
- Runtime configuration is YAML-only, with built-in defaults and validation.
- CI now uses pinned versions of code generation, lint, security scanning, and
  Helm tooling instead of floating `latest` versions.
- Helm defaults now disable OTel when no collector is deployed and apply safer
  non-root Pod security defaults.
- Structured error logging that keeps internal details and panic stack traces
  out of public responses.
- CI now uses a SHA-pinned `setup-helm` action instead of executing a mutable
  remote installer script.
- Server startup now rejects paths that violate the API versioning policy or
  deprecated operations with incomplete/invalid sunset metadata.

### Fixed

- DB-free deployments now report Ready when no database dependency is
  configured; configured database failures still make readiness fail.
- Readiness responses no longer expose raw database driver errors; detailed
  diagnostics remain in server logs.
- Upgraded transitive PostgreSQL and ClickHouse dependencies to versions that
  address the vulnerabilities reported by the pinned security audit.
