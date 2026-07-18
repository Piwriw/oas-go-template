---
name: use-oas-go-template
description: Use when starting a new Go project from the oas-go-template boilerplate — covers running the init-project.sh rename script, regenerating code from OAS spec, swapping in your own API, and avoiding the configuration traps that bite during setup.
---

# Use oas-go-template

## Overview

`oas-go-template` is a Go project template built around OpenAPI Specification 3.x as the single source of truth. It generates gin server stubs and a client SDK from `spec/openapi.yaml` via `oapi-codegen`, ships with config/middleware/otel/version/errcode modules, a Dockerfile, Makefile, golangci-lint config, a Helm chart, and a Vite + React + TS frontend.

This skill tells you how to take the template and turn it into your own project: run the rename script, swap in your API, regenerate code, and avoid the traps that bit during the original build.

## When to Use

Use this skill when:

- You just cloned / copied `oas-go-template` and want to make it yours
- An engineer (or you, in a fresh session) asks "how do I rename the project?" or "how do I change the API?"
- You're onboarding someone to the template and they need a single-page walkthrough

Don't use this skill for:

- Adding endpoints to an existing project derived from this template — just edit `spec/openapi.yaml` and run `make gen`
- General OpenAPI / Go / gin questions

## Files & Their Jobs

| Path | What it does | Will I edit it? |
|------|--------------|-----------------|
| `spec/openapi.yaml` | The contract. Server stubs and client SDK are generated from this. | **Yes — your real API goes here** |
| `scripts/init-project.sh` | One-shot renamer for module path + project name. | Run it once; don't edit |
| `oapi-codegen.yaml` | Generator base config (package name only). | Only if changing package layout |
| `scripts/gen.sh` | Calls oapi-codegen 5 times to produce types + server + client + embedded spec. | No |
| `config.example.yaml` | Sample config; copy to `config.yaml` (gitignored) and edit. | Yes — your real defaults go here |
| `cmd/server/main.go` | Server entrypoint. Wires config → otel → gin → handler. | Rename `serviceName` (auto by init script); otherwise rarely |
| `cmd/client/main.go` | Example client of `pkg/api`. | Optional |
| `internal/api/*.gen.go` | **Generated**. Server types + gin bindings + `StrictServerInterface` + embedded OAS document. | Never hand-edit |
| `internal/handler/` | Your business logic. Implements `StrictServerInterface`. | **Yes — real logic here** |
| `internal/config/` | Loads `config.yaml` and validates. | Add fields as needed |
| `internal/db/` | Gorm init with OTel plugin + connection pool. Disabled when `db.driver` empty. | Yes — register models / add migrations once you adopt it |
| `internal/logging/` | slog setup with trace_id/span_id/request_id injection. | No |
| `internal/otel/` | OTel SDK init (OTLP HTTP traces+metrics, Prometheus pull reader). | No |
| `internal/errcode/` | Typed int32 error codes returned in `api.Error.Code`. | Yes — add codes for your domains |
| `internal/version/` | Holds `Version` / `GitCommit` / `BuildTime` for ldflags. | No |
| `pkg/api/*.gen.go` | **Generated**. Client SDK + client-side types + embedded OAS doc (`GetSpec()` / `GetSpecJSON()`). | Never hand-edit |
| `web/` | Vite + React + TS frontend (independent deploy). | Replace with your UI |
| `build/Dockerfile` | Multi-stage build → static Go binary in alpine. | No |
| `build/otelcol/config.yaml` | Local OTel collector pipeline (OTLP in → Jaeger + debug out). | Tweak exporters if you want Tempo/Prometheus instead |
| `docker-compose.yml` | Local Jaeger + otel-collector for trace verification. | No |
| `chart/` | Helm chart (server + web deploy). | Set image repos by hand after rename |
| `Makefile` | All common commands. | No |
| `.golangci.yml` | Lint config; v2 syntax; excludes `*.gen.go`; forbids legacy `log` package via `forbidigo`. | No |

## The 60-Second Path to "It's Mine"

### Step 1 — Copy the template

```bash
# from where you want the new project to live
cp -r /path/to/oas-go-template ./my-new-project
cd ./my-new-project
rm -rf .git bin client   # drop the template's history + build artifacts
git init
git branch -m main
```

> If you forked on GitHub instead of `cp -r`, the `rm -rf .git && git init` step still applies — you don't want the template's commits as your project history.

### Step 2 — Run the renamer

```bash
./scripts/init-project.sh github.com/yourorg/my-new-project
# or with an explicit short name different from the last path segment:
./scripts/init-project.sh github.com/yourorg/my-new-project my-app
```

The script derives the OLD module path and short name from `go.mod` — it has no hard-coded `"oas-go-template"` string, so it's safe to re-run and won't match itself. It will:

1. Replace `github.com/piwriw/oas-go-template` → your module path across `*.go`, `Makefile`, `build/Dockerfile`, `.golangci.yml`, `scripts/*.sh`, `*.yaml`, `*.yml`, and `*.md` (skipping `*.gen.go`).
2. Replace `oas-go-template` → your short name across the same set, including `cmd/server/main.go`'s `serviceName`, the docker image tags in `Makefile`, the Helm chart (`Chart.yaml`, `templates/_helpers.tpl`, `templates/*.yaml`), and the README/CLAUDE/CONTRIBUTING titles.
3. Run `make gen` so generated code reflects the current spec (idempotent — should produce no diff).
4. Print any leftovers you must handle by hand.

At the end you'll see a "Manual follow-ups" block. **Read it.** It tells you to fix:

- `chart/values.yaml`: `server.image.repository` and `web.image.repository` still say `oas-go-template` / `oas-go-template-web` — set them to your registry paths (the script can't infer your registry).
- `README.md` © line and `chart/Chart.yaml` maintainers — author/copyright info, edit by hand.
- `spec/openapi.yaml` — replace the example `/healthz` `/readyz` `/version` paths with your real API.

### Step 3 — Replace the spec with your real API

Edit `spec/openapi.yaml`. Throw away the `Health` / `VersionInfo` / `Error` examples if you don't need them, but **keep at least one path and one schema** so the generator has something to render. Empty specs produce empty `*.gen.go` files, which then break compilation when `cmd/server/main.go` references symbols that no longer exist.

Then regenerate:

```bash
make gen
```

`scripts/gen.sh` calls `oapi-codegen` **five** times:

| Output file | package | what |
|-------------|---------|------|
| `internal/api/types.gen.go` | `api` | server-side models |
| `internal/api/spec.gen.go` | `api` | gin handlers + `StrictServerInterface` |
| `pkg/api/types.gen.go` | `api` | client-side models (separate copy — `pkg/` cannot import `internal/`) |
| `pkg/api/client.gen.go` | `api` | client SDK |
| `pkg/api/spec.gen.go` | `api` | embedded OAS document — `GetSpec()` / `GetSpecJSON()` for runtime introspection (e.g. serving `/openapi.json`, contract testing) |

### Step 4 — Implement `StrictServerInterface`

Look at the new interface:

```bash
sed -n '/type StrictServerInterface/,/^}/p' internal/api/spec.gen.go
```

For each method, add a file in `internal/handler/`. The `Handler` struct is already declared in `internal/handler/handler.go`. Method signature pattern:

```go
func (h *Handler) GetFoo(ctx context.Context, req api.GetFooRequestObject) (api.GetFooResponseObject, error) {
    return api.GetFoo200JSONResponse(api.Foo{...}), nil
}
```

The response types are `GetFoo200JSONResponse`, `GetFoo500JSONResponse`, etc. — names come from the status code + schema. **Do not invent response types; only use what's in `internal/api/spec.gen.go`.**

`internal/handler/handler_test.go` already pins the contract with a compile-time assertion:

```go
var _ api.StrictServerInterface = (*Handler)(nil)
```

If you forget a method, this line fails the build with a clear error listing every missing method.

### Step 5 — Verify the full pipeline

```bash
make build       # binaries land in bin/
make test        # go test -race -cover ./...
make lint        # golangci-lint v2, excludes *.gen.go
make audit       # govulncheck + gosec (CI gate)
make docker GOPROXY=https://goproxy.cn,direct   # remove GOPROXY if not behind GFW
docker run --rm -d -p 18000:8000 --name smoke my-new-project:latest
curl -sf http://localhost:18000/<your-first-endpoint>
docker stop smoke
```

If `make gen` produced a `git status` diff after this, generation isn't idempotent — investigate before committing.

## What `init-project.sh` Touches (transparency)

If you'd rather do the rename by hand or audit what the script does, here's the full map of where the template's identity lives:

| Reference | Location | Replaced by script? |
|-----------|----------|---------------------|
| Module path | `go.mod:1`, all `*.go` imports, `Makefile` (ldflags), `build/Dockerfile` (ldflags), `.golangci.yml` (`goimports.local-prefixes`), `internal/handler/version.go` (tracer name) | ✓ module pass |
| Short name | `cmd/server/main.go:serviceName`, `Makefile` (docker tags, helm template), `chart/Chart.yaml`, `chart/templates/_helpers.tpl`, `chart/templates/*.yaml`, `chart/NOTES.txt`, `README.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `web/README.md`, `SKILL.md` | ✓ short-name pass |
| Image repository | `chart/values.yaml` (`server.image.repository`, `web.image.repository`) | ✗ manual — registry path isn't derivable |
| Author / copyright | `README.md` (© line), `chart/Chart.yaml` (`maintainers`) | ✗ manual — your name, not the project's |
| Generated code | `internal/api/*.gen.go`, `pkg/api/*.gen.go` | refreshed by `make gen` (skipped by sed) |

The script's `grep` pass uses these include globs: `*.go *.yaml *.yml Makefile Dockerfile *.sh *.md *.tpl *.txt go.mod go.sum`. Anything outside that set won't be touched.

## Make Targets

| Target | What |
|--------|------|
| `make gen` | Regenerate `*.gen.go` from `spec/openapi.yaml` |
| `make build` | Build `bin/server` and `bin/client` (with version ldflags) |
| `make run` | `go run` server (with ldflags) |
| `make run-client` | `go run` client |
| `make test` | `go test -race -cover ./...` |
| `make lint` | `golangci-lint run` (v2) |
| `make fmt` | `goimports` with `-local <module>` to enforce import grouping |
| `make audit` | `govulncheck` + `gosec` (CI gate; non-zero on any finding) |
| `make docker` | Build server image (pass `GOPROXY=...` if behind GFW; passes `VERSION/GIT_COMMIT/BUILD_TIME` via build-arg) |
| `make web-docker` | Build frontend image (multi-stage node → nginx-unprivileged on :8080) |
| `make helm-lint` / `make helm-template` | Validate / render the Helm chart |
| `make web-dev` / `make web-build` | Frontend dev server / production build |
| `make dev-stack` / `make dev-stack-down` | Start / stop local Jaeger + OTel collector |
| `make clean` | Remove `bin/` and `web/dist/` |

## Verifying OTel end-to-end

The template ships `docker-compose.yml` (Jaeger + otel-collector) so you can **prove** the trace pipeline works before relying on it.

```bash
cp config.example.yaml config.yaml   # make sure otel.exporter_otlp_endpoint = http://localhost:4318
make dev-stack                        # start Jaeger + collector
./bin/server                          # or: make run
curl -sf http://localhost:8000/healthz                      # generate a request
# Jaeger UI: http://localhost:16686 → Service = <serviceName>
make dev-stack-down                                         # stop when done
```

What to look for:

- **Log lines** include `trace_id` and `span_id` because `otelgin.Middleware` runs before `logging.Middleware()` in `cmd/server/main.go`. If you swap their order, you lose trace context in logs.
- **Jaeger UI** shows the service with one server span per request.
- **`trace_id` in logs matches the trace ID in Jaeger** — copy-paste to confirm.

If `docker compose up` can't pull images (GFW), pull from a CN mirror and re-tag, or configure a registry mirror in your Docker daemon:

```bash
docker pull docker.1ms.run/jaegertracing/all-in-one:1.60
docker pull docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0
docker tag docker.1ms.run/jaegertracing/all-in-one:1.60 jaegertracing/all-in-one:1.60
docker tag docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0 otel/opentelemetry-collector-contrib:0.110.0
```

To disable OTel entirely (e.g. in unit tests or local dev), set `otel.enabled: false` in `config.yaml` — `otel.Init` returns `(nil, nil)` and the server runs without exporting. `/metrics` still serves Go runtime + process collectors regardless.

## Database (Gorm) — opt-in

`internal/db` ships a Gorm setup with the OTel tracing plugin pre-registered. **Disabled by default** — leave `db.driver` empty in `config.yaml` and the server boots DB-free. Set it + `db.dsn` and `cmd/server/main.go` connects at boot, closes on shutdown.

```yaml
# config.yaml
db:
  driver: postgres                              # postgres | mysql | sqlite
  dsn: "host=localhost user=app password=app dbname=app sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  log_sql: false                                # flip to true to log every SQL statement
```

`*gorm.DB` is already wired into `handler.New(gdb)`. A nil DB means the dependency is intentionally disabled, so `/readyz` reports 200; when DB is configured, handle or ping failures report 503.

## Error codes — `internal/errcode`

`api.Error.Code` is a stable int32 identifier that clients branch on. The OAS schema documents the range allocation; the canonical list lives in `internal/errcode/errcode.go`:

```
10xxx  request / validation
20xxx  auth / authorization
30xxx  not found
50xxx  database / infrastructure
99xxx  internal / unknown
```

When you add domain errors, allocate a range here and define typed constants. Convert to `int32` at the API boundary (`int32(errcode.YourCode)`) — the typed `Code` exists so callers can't pass arbitrary numbers. Never recycle a retired code's number for a new meaning; clients in the wild may still be branching on it.

## Common Mistakes & Traps

These all bit the original build. Read before debugging.

### 1. `oapi-codegen.yaml` v2 syntax

**Wrong** (v1-style, fails to parse):

```yaml
generate:
  - models
  - gin-server
output:
  out: internal/api/spec.gen.go
```

**Right** (v2):

```yaml
package: api
output-options:
  skip-prune: false
```

Leave `generate` and `output` to command-line flags in `scripts/gen.sh`. **If you put `output:` in the config file, it overrides `-o` and all five generations land in the same file.**

### 2. Generated method names don't match tutorials

Older oapi-codegen versions used union wrappers (`GetHealthRes`, `GetHealthOK`). Current versions (v2.7+) use `ResponseObject` interfaces:

```go
GetHealth(ctx context.Context, req GetHealthRequestObject) (GetHealthResponseObject, error)
```

Return a `GetHealth200JSONResponse(...)` / `GetHealth500JSONResponse(...)` value. **Always grep the generated code for the real signature; never assume.**

### 3. Pointer fields for non-required schema properties

```yaml
Health:
  type: object
  required: [status]
  properties:
    version:
      type: string   # not in `required` → pointer in Go
```

Generates `Version *string`, not `Version string`. Either add it to `required` or assign like `v := "0.1.0"; Health{Version: &v}`.

### 4. `go mod tidy` raises the Go directive

You wrote `go 1.23` in `go.mod`. After `go mod tidy`, it becomes `go 1.25.0` because some dependency requires it. This is normal — accept it. Update `build/Dockerfile`'s `FROM golang:X.Y-alpine` to match (otherwise the build fails with `go.mod requires go >= X.Y`).

### 5. semconv version must match the OTel SDK detectors

`internal/otel/otel.go` imports `semconv/v1.41.0`. If you change it to v1.26.0 "because that's what tutorials use", `resource.New(...)` fails at startup with:

```
otel init: otel resource: error detecting resource: conflicting Schema URL: https://opentelemetry.io/schemas/1.26.0 and https://opentelemetry.io/schemas/1.41.0
```

Use whatever version ships with your `go.opentelemetry.io/otel` major version. Check with:

```bash
ls $(go env GOMODCACHE)/go.opentelemetry.io/otel@$(go list -m -f '{{.Version}}' go.opentelemetry.io/otel)/semconv/
```

Pick the highest `v1.X.0` directory. That's what the SDK's detectors use.

### 6. golangci-lint v2 config syntax

v1 uses `linters.enable: [...]` (flat) and `issues.exclude-rules`. v2 reshuffles both:
- Linters: `linters.default: standard` + `linters.enable: [...]` (nested).
- Exclusions moved **out of `issues`** into `linters.exclusions`. For generated code, prefer `linters.exclusions.generated: strict` (auto-skips any file with the `// Code generated by ... DO NOT EDIT.` header) over hand-maintained path regex.
- `issues` itself lost `exclude-rules` entirely — only `max-issues-per-linter`, `max-same-issues`, `new-*`, and a few others remain.

The repo's `.golangci.yml` is v2 and passes `golangci-lint config verify`. If you downgrade to v1, rewrite the file.

### 7. Stale server process makes you think ldflags didn't work

`go run ./cmd/server` leaves a cached binary in `$GOCACHE` that may keep running after you Ctrl-C. When you then `curl /version` and see `dev/unknown/unknown` despite correct ldflags, the old process is still bound to :8000.

Before debugging ldflags, run:

```bash
lsof -ti:8000 | xargs -r kill -9
```

### 8. Docker build fails on `go mod download` behind GFW

Container can't reach `proxy.golang.org`. Pass a CN-friendly mirror:

```bash
make docker GOPROXY=https://goproxy.cn,direct
```

The Makefile and Dockerfile already forward this through `--build-arg GOPROXY=...`.

### 9. `os.Exit(0)` skips defers — golangci-lint will catch it

If you reach for `os.Exit(0)` at the end of `main`, gocritic flags `exitAfterDefer`. Just let `main` return — process exits with status 0.

### 10. Generated code must be checked in, not gitignored

`*.gen.go` files are committed to git. They are stable across runs (`make gen` is idempotent — verified by `git status` being clean afterwards). Do **not** add `*.gen.go` to `.gitignore`; reviewers and IDEs need to see the actual code being compiled.

### 11. Middleware order: `otelgin` BEFORE `logging`

`logging.Middleware()` reads the active span from `c.Request.Context()` to inject `trace_id` / `span_id` into each log record. `otelgin.Middleware` is what puts the span there. Reversed order → no trace context in logs, and you'll be debugging "where did trace_id go?" for an hour.

The correct chain in `cmd/server/main.go`:

```go
r.Use(gin.Recovery(), otelgin.Middleware(serviceName), logging.Middleware())
```

### 12. `docker compose up` can't pull images (GFW)

If `make dev-stack` fails with `registry-1.docker.io` timeouts, configure a Docker registry mirror, or pre-pull from a CN mirror and re-tag (see the "Verifying OTel end-to-end" section above for exact commands).

### 13. SQLite `:memory:` is per-connection

Each connection to `file::memory:` gets its own private database. With a connection pool, your migration lands on connection A, the next query runs on connection B which sees an empty DB. Fix: use `file::memory:?cache=shared` **and** set `DB_MAX_OPEN_CONNS=1`. The `internal/db/db_test.go` test does exactly this.

### 14. Pass `*gorm.DB` via the handler constructor

Don't reach for a package-level global `db.DB`. The template already wires `*gorm.DB` through `handler.New(gdb)`; keep that pattern. Keeps tests able to swap a sqlite memory DB.

### 15. Missing `config.yaml` is **not** an error

`config.Load` falls back to built-in defaults when the file is absent — that way tests and scratch runs work without authoring a config. If your prod deployment requires the file (e.g. you don't want to silently boot with defaults), check existence yourself before calling `Load`, or fail validation in `internal/config.validate(...)` for envs where the defaults are unsafe.

### 16. Don't use the `log` package — `log/slog` only

The repo's `.golangci.yml` enables `forbidigo` with `analyze-types: true` and anchored patterns `^log\.(Print|Fatal|Panic)(ln|f)?$`. The legacy `log` package is forbidden; use `log/slog` for all logging. For process exit on error, `slog.Error(...)` + `os.Exit(1)` instead of `log.Fatalf`.

### 17. The `client` binary in repo root is a build artifact

`go build ./cmd/client` from the repo root drops a `client` binary in `.`. Don't commit it. `make build` correctly puts binaries under `bin/`; running bare `go build` doesn't.

## Daily Workflow Once Renamed

```bash
# Edit spec/openapi.yaml, then:
make gen
# Implement new methods in internal/handler/
make lint test
git add spec/ internal/api/ pkg/api/ internal/handler/
git commit -m "feat(api): add /foo endpoint"
```

## Frontend ↔ Backend Contract

The frontend (`web/`) is **independent** — server doesn't serve its static files. If you want a typed TypeScript client matching `spec/openapi.yaml`, the template intentionally leaves `web/src/api/` empty. Add this yourself with `openapi-typescript` and/or `openapi-fetch`:

```bash
cd web
npm install openapi-fetch openapi-typescript
npx openapi-typescript ../spec/openapi.yaml -o src/api/schema.gen.ts
```

(Not part of the template — left as an exercise because everyone's frontend stack differs.)

## License

The template ships with an MIT `LICENSE`. Replace it with your own license (MIT, Apache-2.0, etc.) before publishing, and update the © line in `README.md` (the `init-project.sh` script flags this as a manual follow-up since it can't infer the new copyright holder).
