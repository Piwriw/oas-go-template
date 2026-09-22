# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## What this is

`oas-go-template` is a Go project template where **`spec/openapi.yaml` is the single source of truth**. Server stubs and the client SDK are generated from it via `oapi-codegen` (v2 StrictServerInterface mode). All other code (config, otel, logging, db, service, handlers) supports that contract.

For "how to derive a new project from this template" see `SKILL.md`. AGENTS.md is for working **inside** the repo.

## Commands

| Task | Command |
|------|---------|
| Regenerate backend code from `spec/openapi.yaml` | `make gen` |
| Regenerate frontend API client | `make gen-web` |
| Regenerate both | `make gen-all` |
| Build server | `make build` |
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

`scripts/gen.sh` invokes the generator pinned in `tools/go.mod` five times
against `spec/openapi.yaml`. The `spec/*.cfg.yaml` files select models, server,
client, or embedded-spec generation; the script owns input and output paths.
`models.cfg.yaml` is shared by the two model outputs:

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
than the deprecation date. Keep the operation available until sunset; removing
it earlier is a breaking contract change.

`make contract-check BASE_SPEC=/path/to/openapi-base.yaml` runs pinned
`oasdiff` v1.10.28. Pull request CI supplies the target branch's spec as the
baseline and fails on ERR-level breaking changes. Intentional breaking changes
require a new `/vN` API version and a migration plan.

### StrictServerInterface pattern

Handlers in `internal/handler/` implement `api.StrictServerInterface` — a generated interface where each method returns a typed `ResponseObject` (`GetFoo200JSONResponse`, `GetFoo500JSONResponse`, etc.). The constructor `api.NewStrictHandler(h, nil)` wraps them; `api.RegisterHandlers(r, strictHandler)` mounts them on gin. There is a compile-time check `var _ api.StrictServerInterface = (*Handler)(nil)` in `internal/handler/handler_test.go` so missing methods fail the build.

Business logic lives in `internal/service` (`service.New(gdb)` wired via `handler.New(svc)`); handlers are thin adapters that map service results and sentinel errors to generated response types — `errcode` mapping happens only there because codes are part of the public API contract. HTTP error plumbing (`Recovery`, `BodyLimit`, `StrictHandlerOptions`, `NoRoute`/`NoMethod`, `OAPIValidationError`, `writeError`) lives in `internal/middleware`, which must not import `internal/handler`.

Response type names come from the OAS status code + schema — **only use names that already exist in `internal/api/spec.gen.go`**, never invent them.

### Request lifecycle and middleware ordering

`cmd/server/main.go:newHTTPServer` calls `middleware.Use`, which wires the
built-in chain in this order:

```go
middleware.Use(r, middleware.Options{
    ServiceName: serviceName, MaxBodyBytes: maxRequestBodyBytes,
})
```

That expands to recovery, OTel, logging, CORS, and body limit in that
order.

CORS is code-owned rather than configurable. It always allows every origin
without credentials, permits GET/POST/PUT/PATCH/DELETE/OPTIONS, exposes the
request ID response header, and caches preflight responses for 12 hours.

HTTP timeouts, header size, and request-body size are fixed constants in
`cmd/server/main.go`; they are intentionally not part of `config.yaml`.

Generated API routes add the embedded OAS request validator and use
`middleware.StrictHandlerOptions()` for the common `api.Error` response. The
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

After the ping, `Init` runs embedded SQL migrations from
`internal/db/migrations/`. Each change is a
`YYYYMMDDHHMMSS_name.up.sql` / `.down.sql` pair managed by `golang-migrate`.
The current version and dirty state live in `schema_migrations`; applied
versions are skipped. Never edit or reuse an applied version.

Migrations can also run manually without booting the server: `make migrate-up`
applies every pending migration and `make migrate-down` rolls back exactly one
version against the DSN in `CONFIG` (default `config.yaml`). Both targets invoke
the dedicated `cmd/migrate` entrypoint.

`*gorm.DB` is injected via `service.New(gdb)` and the service via `handler.New(svc)`; **`db` may be nil** when the dependency is intentionally disabled, and `/readyz` reports 200 in that case. Use the same pattern for any new optional dependency.

### /healthz vs /readyz

Two separate probes — `internal/service/health.go` owns the readiness rule, `internal/handler/health.go` maps it to HTTP:
- `GET /healthz` — **liveness**. 200 as long as the process is up; returns real `version.Version`.
- `GET /readyz` — **readiness**. 200 when all configured deps are reachable; a disabled DB is skipped, while a configured DB ping failure returns 503. Don't add expensive checks to `/healthz`.

Graceful shutdown calls `http.Server.Shutdown` with a 10s deadline. It closes
listeners immediately and waits for in-flight requests; readiness has no
separate shutdown state or drain delay. Keep the Helm
`terminationGracePeriodSeconds` longer than the shutdown deadline.

### /metrics

`GET /metrics` is hardcoded in `cmd/server/main.go:newHTTPServer` and serves `promhttp.Handler()` from `prometheus.DefaultGatherer`. Always on, not configurable — it's an ops endpoint, not part of the API contract, and there's no good reason to disable it. Intentionally absent from `spec/openapi.yaml` so the client SDK doesn't carry a useless `GetMetrics*` method. It is routed through the full middleware chain and traced. For `/metrics`, `/healthz`, and `/readyz`, `logging.Middleware` records the first successful request and every error, then suppresses repeated successes to avoid scrape and probe noise.

### Version injection

`internal/version/version.go` holds `Version` / `GitCommit` / `BuildTime` populated via `-ldflags -X` (see `Makefile:LDFLAGS` and `build/Dockerfile`). `go run` skips ldflags → empty fields → `internal/service/version.go` degrades to `"dev"` / `"unknown"` so `/version` still returns 200 instead of 500. Don't error on empty version fields.

### Frontend is independent

`web/` (Next.js + React + TS) deploys separately from the server as a static export. `web/Dockerfile` is multi-stage (node build → nginx-unprivileged on `:8080`); backend runs on `:8000`. The server does **not** serve `web/out`.

`make gen-web` generates `web/src/api/schema.gen.ts` from the same `spec/openapi.yaml` using `openapi-typescript` (pinned in `web/package.json`; run `npm ci` in `web/` first). That file is committed like `*.gen.go` — **never hand-edit it** — and carries *types only*. The runtime client is the hand-written `web/src/api/client.ts`, which owns the `createClient<paths>()` instance from `openapi-fetch`; unlike a generator that emits the client for you, that file is yours to edit (base URL, middleware, auth). `openapi-typescript`'s TypeScript peer is `^5.x`, so `web/` is pinned to TS 5.x — bumping `web/` to TS 6 makes `npm ci` fail with `ERESOLVE`.

### Developer tool management

The `tool` directive in `tools/go.mod` pins `oapi-codegen`, isolating generator
dependencies from the application. `make gen` runs it from that module with
absolute input and output paths. The root `go.mod` still pins `goimports` and
`govulncheck`; invoke these tools through `go tool`, not globally installed binaries.
After dependency changes, tidy both modules with `go mod tidy` and
`go -C tools mod tidy`. CI checks both modules and regenerates Go and TypeScript
outputs with `make gen-all` to detect drift from the spec or generator configs.
`golangci-lint` intentionally stays outside the application module graph:
local development uses its official release binary and CI uses the official
Action pinned to a commit SHA. `oasdiff` and `gosec` remain isolated behind
versioned `go run` commands. `make tools` only installs the pinned `air` binary
for optional local live reload.

## Watch-outs

- **Test behavior, not plumbing**: do not add unit tests that merely re-verify Go standard-library or third-party behavior, or straightforward field-to-option assignments. For configuration switches, cover built-in defaults and explicit YAML overrides at the `config.Load` boundary; add deeper behavior tests only when the project implements custom branching, transformation, or failure handling.
- **Database code needs no tests**: `internal/db/` and its subpackages (`models/`, `store/`) carry no unit tests — do not add or restore test files there. The package is thin plumbing over gorm and golang-migrate; reachability is covered by `/readyz`, and DB behavior is verified against real environments instead of unit tests.
- **Function comments**: every named function and method in non-generated Go code, including tests and test helpers, must have exactly one concise comment line immediately above its declaration. Use `// FunctionName ...`, start with the exact function name, and describe the concrete business responsibility rather than restating the signature. Anonymous functions are exempt; never edit `*.gen.go` to add comments.
- **Database model field comments**: every field in a non-generated persistent database model must have a concise comment line immediately above the field declaration. The comment must describe the field's business meaning; trailing comments do not satisfy this requirement. Never edit `*.gen.go` to add these comments.
- **Named constants without over-extraction**: values that are reused or define business/protocol invariants (route paths, context keys, header names, etc.) belong in a named `const` block immediately after the imports. Do not extract one-off SQL fragments, column names, sort expressions, or other local implementation details merely to avoid literals; keep them at the call site and prefer typed library APIs that eliminate repeated strings. Repeated values used across `switch` cases or conditionals still require named constants.
- **golangci-lint v2 config syntax** (`.golangci.yml`): uses `default: standard` + `enable: [...]`, not v1's flat `enable`. Generated code is excluded via `path: '.*\.gen\.go$'`.
- **`os.Exit(0)` after defers**: gocritic's `exitAfterDefer` will fail lint. `main` returns through `run()` and exits via `os.Exit(1)` only on error — keep it that way.
- **Empty OAS spec breaks the build**: keep at least one path and one schema in `spec/openapi.yaml`, otherwise `cmd/server/main.go` references symbols that no longer exist after `make gen`.
- **OTel `semconv` version is pinned to v1.41.0** for a reason (matches SDK detectors — bumping it crashes resource init with a conflicting Schema URL error). Update only when you also bump `go.opentelemetry.io/otel`.
