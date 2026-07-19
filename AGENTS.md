# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## What this is

`oas-go-template` is a Go project template where **`spec/openapi.yaml` is the single source of truth**. Server stubs and the client SDK are generated from it via `oapi-codegen` (v2 StrictServerInterface mode). All other code (config, otel, logging, db, handlers) supports that contract.

For "how to derive a new project from this template" see `SKILL.md`. AGENTS.md is for working **inside** the repo.

## Commands

| Task | Command |
|------|---------|
| Regenerate code from `spec/openapi.yaml` | `make gen` |
| Build server + client | `make build` |
| Run server (with ldflags) | `make run` |
| Run all tests | `make test` |
| Run a single test | `go test -run TestLoad_fullYAML ./internal/config` |
| Lint (golangci-lint v2) | `make lint` |
| Format (goimports, three-group) | `make fmt` |
| Security audit (govulncheck + gosec) | `make audit` |
| OpenAPI breaking-change check | `make contract-check BASE_SPEC=...` |
| Supply-chain pin check | `make supply-chain-check` |
| Build server Docker image | `make docker` |
| Build frontend Docker image | `make web-docker` |
| Local Jaeger + OTel collector | `make dev-stack` / `make dev-stack-down` |

`audit` exits non-zero on any reachable vuln or finding; that's intentional for CI. `fmt` enforces std / third-party / `github.com/piwriw/oas-go-template` ordering via `-local`. The repository toolchain is Go `1.26.5`; `make supply-chain-check` verifies that version, explicit Docker tags, and GitHub Action SHAs remain aligned.

## Architecture

### OAS-driven codegen (5 outputs, one spec)

`scripts/gen.sh` invokes `oapi-codegen` five times against `spec/openapi.yaml`:

| Output | Package | Role |
|--------|---------|------|
| `internal/api/types.gen.go` | `api` | server-side models |
| `internal/api/spec.gen.go` | `api` | gin bindings + `StrictServerInterface` |
| `pkg/api/types.gen.go` | `api` | client-side models (separate copy — `pkg/` cannot import `internal/`) |
| `pkg/api/client.gen.go` | `api` | client SDK |
| `pkg/api/spec.gen.go` | `api` | embedded OAS document — `GetSpec()` / `GetSpecJSON()` for runtime introspection (e.g. serving `/openapi.json`, contract testing) |

`pkg/` mirrors `internal/` because Go's import visibility prevents the public client from importing the server's types — both copies must exist. The embedded-spec file pulls in `github.com/getkin/kin-openapi` as a direct dep.

**Never hand-edit `*.gen.go`.** They are committed (not gitignored) so reviewers and IDEs see what's compiled.

### API versioning and deprecation

`spec/openapi.yaml` declares the API policy with `x-api-version: v1` and
`x-versioning.strategy: url-prefix`. `/healthz`, `/readyz`, and `/version` are
explicit unversioned operational exceptions; every future business path must
use a `/vN/` prefix. `internal/oas` validates this policy at startup.

For a deprecated operation, set `deprecated: true` and provide RFC3339
`x-deprecation-date` and `x-sunset-date` extensions. The sunset must be later
than the deprecation date. The global middleware resolves the matched Gin route
against the embedded OAS operation and emits `Deprecation` and `Sunset`
response headers. Keep the operation available until sunset; removing it
earlier is a breaking contract change.

`make contract-check BASE_SPEC=/path/to/openapi-base.yaml` runs pinned
`oasdiff` v1.10.28. Pull request CI supplies the target branch's spec as the
baseline and fails on ERR-level breaking changes. Intentional breaking changes
require a new `/vN` API version and a migration plan.

### StrictServerInterface pattern

Handlers in `internal/handler/` implement `api.StrictServerInterface` — a generated interface where each method returns a typed `ResponseObject` (`GetFoo200JSONResponse`, `GetFoo500JSONResponse`, etc.). The constructor `api.NewStrictHandler(h, nil)` wraps them; `api.RegisterHandlers(r, strictHandler)` mounts them on gin. There is a compile-time check `var _ api.StrictServerInterface = (*Handler)(nil)` in `internal/handler/handler_test.go` so missing methods fail the build.

Response type names come from the OAS status code + schema — **only use names that already exist in `internal/api/spec.gen.go`**, never invent them.

### Request lifecycle and middleware ordering

`cmd/server/main.go:newHTTPServer` calls `middleware.Use`, which wires the
built-in chain in this order:

```go
middleware.Use(r, middleware.Options{
    ServiceName: serviceName, MaxBodyBytes: cfg.Server.MaxBodyBytes,
    CORS: cfg.CORS, OpenAPISpec: spec,
})
```

That expands to recovery, OTel, logging, optional CORS, body limit, and
optional OAS deprecation headers in that order.

Generated API routes add the embedded OAS request validator and use
`handler.StrictServerOptions()` for the common `api.Error` response. The
operational `/metrics` route remains outside the OAS validator group.

`otelgin` must run **before** `logging.Middleware` — logging reads the active span from `c.Request.Context()` to inject `trace_id` / `span_id` into each slog record (see `internal/logging/logging.go:otelHandler`). Reverse the order and trace context silently disappears from logs.

### Config loading

`internal/config/config.go:Load` merges in this order:
1. Built-in `defaults()` (HTTPAddr `:8000`, GinMode `debug`, OTel enabled, pool sizes, etc.)
2. `config.yaml` (path from `-c` flag, default `config.yaml`)

There is **no env-var overlay** — YAML is the only source. Missing file is OK (defaults take over); any other stat/read error is returned. `validate()` runs after the merge (`gin_mode` whitelist, `log.format`, `db.driver` whitelist + DSN-required-when-driver-set, etc.).

`config.yaml` is gitignored; commit only `config.example.yaml`.

### OTel init

`internal/otel/otel.go:Init` sets up TracerProvider + MeterProvider with two MeterProvider readers: an OTLP HTTP periodic reader (push) and an OTel Prometheus exporter (pull, fed into `prometheus.DefaultRegisterer`). Uses `semconv/v1.41.0` (must match the OTel SDK's bundled detectors — see SKILL.md trap #5). Returns `(nil, nil)` when `cfg.Enabled=false`. `cmd/server/main.go:run` defers `shutdownOTel` so exporter flush happens on signal.

When OTel is disabled, `/metrics` still serves Go runtime + process collectors (auto-registered by `prometheus/client_golang`'s `init`). When enabled, the same registry also carries OTel-translated app metrics (DB spans, gin server metrics, etc., depending on instrumentation wired in).

### DB (Gorm, opt-in)

`internal/db/db.go:Init` returns `(nil, nil)` when `cfg.DB.Driver` is empty — server boots DB-free. When set, it opens postgres/mysql/sqlite, registers `gorm.io/plugin/opentelemetry` (every SQL op becomes a child span), and pings with a 5s timeout.

`*gorm.DB` is injected via `handler.New(gdb)`; **`db` may be nil** when the dependency is intentionally disabled, and `/readyz` reports 200 in that case. Use the same pattern for any new optional dependency.

For sqlite tests use `file::memory:?cache=shared` + `DB_MAX_OPEN_CONNS=1` — see `internal/db/db_test.go`. With a connection pool, each connection otherwise gets its own private memory DB.

### /healthz vs /readyz

Two separate probes in `internal/handler/health.go`:
- `GET /healthz` — **liveness**. 200 as long as the process is up; returns real `version.Version`.
- `GET /readyz` — **readiness**. 200 when all configured deps are reachable; a disabled DB is skipped, while a configured DB ping failure returns 503. Don't add expensive checks to `/healthz`.

During graceful shutdown, `handler.DrainState` flips readiness to 503 before
`http.Server.Shutdown` begins. `server.drain_timeout` defaults to 5s so load
balancers can observe the state change; keep the Helm
`terminationGracePeriodSeconds` longer than that window.

### /metrics

`GET /metrics` is hardcoded in `cmd/server/main.go:newHTTPServer` and serves `promhttp.Handler()` from `prometheus.DefaultGatherer`. Always on, not configurable — it's an ops endpoint, not part of the API contract, and there's no good reason to disable it. Intentionally absent from `spec/openapi.yaml` so the client SDK doesn't carry a useless `GetMetrics*` method. Routed through the full middleware chain (otelgin + logging) — every scrape is traced and logged.

### Version injection

`internal/version/version.go` holds `Version` / `GitCommit` / `BuildTime` populated via `-ldflags -X` (see `Makefile:LDFLAGS` and `build/Dockerfile`). `go run` skips ldflags → empty fields → `internal/handler/version.go` degrades to `"dev"` / `"unknown"` so `/version` still returns 200 instead of 500. Don't error on empty version fields.

### Frontend is independent

`web/` (Vite + React + TS) deploys separately from the server. `web/Dockerfile` is multi-stage (node → nginx-unprivileged on `:8080`); backend runs on `:8000`. The server does **not** serve `web/dist`. There is no typed client generated into `web/src/api/` — that's intentional (left for the user's stack choice).

## Watch-outs

- **golangci-lint v2 config syntax** (`.golangci.yml`): uses `default: standard` + `enable: [...]`, not v1's flat `enable`. Generated code is excluded via `path: '.*\.gen\.go$'`.
- **`os.Exit(0)` after defers**: gocritic's `exitAfterDefer` will fail lint. `main` returns through `run()` and exits via `os.Exit(1)` only on error — keep it that way.
- **Empty OAS spec breaks the build**: keep at least one path and one schema in `spec/openapi.yaml`, otherwise `cmd/server/main.go` references symbols that no longer exist after `make gen`.
- **OTel `semconv` version is pinned to v1.41.0** for a reason (matches SDK detectors — bumping it crashes resource init with a conflicting Schema URL error). Update only when you also bump `go.opentelemetry.io/otel`.
