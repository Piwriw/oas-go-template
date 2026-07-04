---
name: use-oas-go-template
description: Use when starting a new Go project from the oas-go-template boilerplate — covers cloning, renaming module path, regenerating code from OAS spec, swapping in your own API, and avoiding the configuration traps that bite during setup.
---

# Use oas-go-template

## Overview

`oas-go-template` is a Go project template built around OpenAPI Specification 3.x as the single source of truth. It generates gin server stubs and a client SDK from `spec/openapi.yaml` via `oapi-codegen`, ships with config/middleware/otel/version modules, a Dockerfile, Makefile, golangci-lint config, and a Vite + React + TS frontend.

This skill tells you how to take the template and turn it into your own project: rename the module, swap in your API, regenerate code, and avoid the traps that bit during the original build.

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
| `oapi-codegen.yaml` | Generator base config (package name only). | Only if changing package layout |
| `scripts/gen.sh` | Calls oapi-codegen 4 times to produce types + server + client code. | No |
| `config.example.yaml` | Sample config; copy to `config.yaml` (gitignored) and edit. | Yes — your real defaults go here |
| `cmd/server/main.go` | Server entrypoint. Wires config → otel → gin → handler. | Rename `serviceName` const; otherwise rarely |
| `cmd/client/main.go` | Example client of `pkg/api`. | Optional |
| `internal/api/*.gen.go` | **Generated**. Server types + gin bindings + `StrictServerInterface`. | Never hand-edit |
| `internal/handler/` | Your business logic. Implements `StrictServerInterface`. | **Yes — real logic here** |
| `internal/middleware/` | Custom gin middleware (logger). | Add more here |
| `internal/config/` | Loads `config.yaml` and validates. | Add fields as needed |
| `internal/db/` | Gorm init with OTel plugin + connection pool. Disabled when `db.driver` empty. | Yes — register models / add migrations once you adopt it |
| `internal/logging/` | slog setup with trace_id/span_id/request_id injection. | No |
| `internal/otel/` | OTel SDK init (OTLP HTTP traces+metrics). | No |
| `internal/version/` | Holds `Version` / `GitCommit` / `BuildTime` for ldflags. | No |
| `pkg/api/*.gen.go` | **Generated**. Client SDK + client-side types. | Never hand-edit |
| `web/` | Vite + React + TS frontend (independent deploy). | Replace with your UI |
| `build/Dockerfile` | Multi-stage build → static Go binary in alpine. | No |
| `build/otelcol/config.yaml` | Local OTel collector pipeline (OTLP in → Jaeger + debug out). | Tweak exporters if you want Tempo/Prometheus instead |
| `docker-compose.yml` | Local Jaeger + otel-collector for trace verification. | No |
| `Makefile` | All common commands. | No |
| `.golangci.yml` | Lint config; excludes `*.gen.go`. | No |

## The 5-Minute Path to "It's Mine"

### Step 1 — Copy the template

```bash
# from where you want the new project to live
cp -r /path/to/oas-go-template ./my-new-project
cd ./my-new-project
rm -rf .git
git init
git branch -m main
```

### Step 2 — Rename the module path

The template's module is `github.com/piwriw/oas-go-template`. Replace every occurrence with your own. Example target: `github.com/yourorg/my-new-project`.

```bash
OLD=github.com/piwriw/oas-go-template
NEW=github.com/yourorg/my-new-project

# 1) go.mod
sed -i.bak "s|^module $OLD|module $NEW|" go.mod && rm go.mod.bak

# 2) every Go import
grep -rl "$OLD" --include='*.go' . \
  | grep -v '\.gen\.go$' \
  | xargs sed -i.bak "s|$OLD|$NEW|g"
find . -name '*.bak' -delete

# 3) Makefile (ldflags -X paths reference the module)
sed -i.bak "s|$OLD|$NEW|g" Makefile && rm Makefile.bak

# 4) Dockerfile (ldflags -X paths there too)
sed -i.bak "s|$OLD|$NEW|g" build/Dockerfile && rm build/Dockerfile.bak

# 5) scripts/gen.sh
sed -i.bak "s|$OLD|$NEW|g" scripts/gen.sh && rm scripts/gen.sh.bak
```

**Do NOT skip `*.gen.go`** — those are regenerated in Step 4 and would otherwise drift.

### Step 3 — Rename `serviceName` in `cmd/server/main.go`

```go
const serviceName = "oas-go-template"  // ← change to "my-new-project"
```

This is what OTel sends as `service.name`. Pick a stable, lowercase, hyphen-free-of-spaces identifier.

### Step 4 — Replace the spec with your real API

Edit `spec/openapi.yaml`. Throw away the `Health` / `VersionInfo` / `Error` examples if you don't need them, but **keep at least one path and one schema** so the generator has something to render. Empty specs produce empty `*.gen.go` files, which then break compilation when `cmd/server/main.go` references symbols that no longer exist.

### Step 5 — Regenerate

```bash
make gen
```

This runs `scripts/gen.sh`, which calls `oapi-codegen` four times:

| Output file | package | what |
|-------------|---------|------|
| `internal/api/types.gen.go` | `api` | server-side models |
| `internal/api/server.gen.go` | `api` | gin handlers + `StrictServerInterface` |
| `pkg/api/types.gen.go` | `api` | client-side models (separate copy — `pkg/` cannot import `internal/`) |
| `pkg/api/client.gen.go` | `api` | client SDK |

### Step 6 — Implement `StrictServerInterface`

Look at the new interface:

```bash
sed -n '/type StrictServerInterface/,/^}/p' internal/api/server.gen.go
```

For each method, write a file in `internal/handler/`. The Handler struct is already declared in `internal/handler/handler.go`. Method signature pattern:

```go
func (h *Handler) GetFoo(ctx context.Context, req api.GetFooRequestObject) (api.GetFooResponseObject, error) {
    return api.GetFoo200JSONResponse(api.Foo{...}), nil
}
```

The response types are `GetFoo200JSONResponse`, `GetFoo500JSONResponse`, etc. — their names come from the status code + schema. **Do not invent response types; only use what's in `internal/api/server.gen.go`.**

Add a compile-time check so you can't forget a method:

```go
// internal/handler/handler_test.go
package handler

import (
    "testing"
    "github.com/yourorg/my-new-project/internal/api"
)

var _ api.StrictServerInterface = (*Handler)(nil)

func TestNewReturnsHandler(t *testing.T) {
    if New() == nil { t.Fatal("nil handler") }
}
```

### Step 7 — Verify the full pipeline

```bash
make build    # binaries land in bin/
make test
make lint     # golangci-lint, excludes *.gen.go
make docker GOPROXY=https://goproxy.cn,direct   # remove GOPROXY if not behind GFW
docker run --rm -d -p 18000:8000 --name smoke oas-go-template:latest  # rename tag in Makefile if you care
curl -sf http://localhost:18000/<your-first-endpoint>
docker stop smoke
```

If `make gen` produced a `git status` diff after this, generation isn't idempotent — investigate before committing.

### Step 8 — Rename the Helm chart (if you kept it)

`chart/` still references the old project name. Step 2's sed loop only touches
`*.go`, `Makefile`, `build/Dockerfile`, and `scripts/gen.sh` — the chart needs
its own pass.

```bash
NEW=my-new-project   # last segment of your module path

# 1) Chart name + helpers + README + NOTES reference "oas-go-template".
grep -rl 'oas-go-template' chart/ \
  | xargs sed -i.bak "s/oas-go-template/${NEW}/g"
find chart -name '*.bak' -delete

# 2) Image repositories in chart/values.yaml — these are registry paths, not
#    derivable from the module name. Edit by hand:
#      server.image.repository: oas-go-template       → <registry>/${NEW}
#      web.image.repository:    oas-go-template-web   → <registry>/${NEW}-web

# 3) Validate.
make helm-lint
make helm-template
```

Skip this entirely if you don't deploy via Helm — `rm -rf chart/` is fine; the
rest of the project (build, test, lint) doesn't reference it.

## Make Targets

| Target | What |
|--------|------|
| `make gen` | Regenerate `*.gen.go` from `spec/openapi.yaml` |
| `make build` | Build `bin/server` and `bin/client` (with version ldflags) |
| `make run` | `go run` server (with ldflags) |
| `make run-client` | `go run` client |
| `make test` | `go test -race -cover ./...` |
| `make lint` | `golangci-lint run` |
| `make docker` | Build server image (pass `GOPROXY=...` if behind GFW; passes `VERSION/GIT_COMMIT/BUILD_TIME` automatically) |
| `make web-dev` / `make web-build` | Frontend |
| `make dev-stack` / `make dev-stack-down` | Start / stop local Jaeger + OTel collector |
| `make clean` | Remove `bin/` and `web/dist/` |

## Verifying OTel end-to-end

The template ships a `docker-compose.yml` (Jaeger + otel-collector) so you can
**prove** the trace pipeline works before relying on it.

```bash
cp config.example.yaml config.yaml   # make sure otel.exporter_otlp_endpoint = http://localhost:4318
make dev-stack                        # start Jaeger + collector
./bin/server                          # or: make run
curl -sf http://localhost:8000/healthz                      # generate a request
# Jaeger UI: http://localhost:16686 → Service = <serviceName>
make dev-stack-down                                         # stop when done
```

What to look for:

- **Log lines** include `trace_id` and `span_id` because `otelgin.Middleware`
  runs before `logging.Middleware()` in `cmd/server/main.go`. If you swap their
  order, you lose trace context in logs.
- **Jaeger UI** shows the service with one server span per request plus any
  child spans the handler creates (e.g. `internal/handler/version.go` opens a
  manual `Handler.GetVersion` span).
- **`trace_id` in logs matches the trace ID in Jaeger** — copy-paste to confirm.

If `docker compose up` can't pull images (GFW), pull from a CN mirror and
re-tag, or configure a registry mirror in your Docker daemon:

```bash
docker pull docker.1ms.run/jaegertracing/all-in-one:1.60
docker pull docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0
docker tag docker.1ms.run/jaegertracing/all-in-one:1.60 jaegertracing/all-in-one:1.60
docker tag docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0 otel/opentelemetry-collector-contrib:0.110.0
```

To disable OTel entirely (e.g. in unit tests or local dev), set
`otel.enabled: false` in `config.yaml` — `otel.Init` returns `(nil, nil)` and
the server runs without exporting.

## Database (Gorm) — opt-in

`internal/db` ships a Gorm setup with the OTel tracing plugin pre-registered.
**Disabled by default** — leave `db.driver` empty in `config.yaml` and the
server boots DB-free. Set it + `db.dsn` and `cmd/server/main.go` connects at
boot, closes on shutdown.

```yaml
# config.yaml
db:
  driver: postgres                              # postgres | mysql | sqlite
  dsn: "host=localhost user=app password=app dbname=app sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
```

The handler doesn't yet take `*gorm.DB` — when you adopt it, extend
`handler.New()` to accept a `*gorm.DB` (or an interface) and pass it from
`cmd/server/main.go`. Stash it on the `Handler` struct.

## Common Mistakes & Traps

These all bit the original build. Read before debugging.

### 1. `oapi-codegen.yaml` v2 syntax

**Wrong** (v1-style, fails to parse):

```yaml
generate:
  - models
  - gin-server
output:
  out: internal/api/server.gen.go
```

**Right** (v2):

```yaml
package: api
output-options:
  skip-prune: false
```

Leave `generate` and `output` to command-line flags in `scripts/gen.sh`. **If you put `output:` in the config file, it overrides `-o` and all four generations land in the same file.**

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

v1 uses `linters.enable: [...]`. v2 uses `linters.default: standard` + `linters.enable: [...]`. The repo's `.golangci.yml` is v2 — if you downgrade to v1, rewrite the file.

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

`logging.Middleware()` reads the active span from `c.Request.Context()` to inject
`trace_id` / `span_id` into each log record. `otelgin.Middleware` is what puts
the span there. Reversed order → no trace context in logs, and you'll be
debugging "where did trace_id go?" for an hour.

The correct chain in `cmd/server/main.go`:

```go
r.Use(gin.Recovery(), otelgin.Middleware(serviceName), logging.Middleware())
```

### 12. `docker compose up` can't pull images (GFW)

If `make dev-stack` fails with `registry-1.docker.io` timeouts, configure a
Docker registry mirror, or pre-pull from a CN mirror and re-tag (see the
"Verifying OTel end-to-end" section above for exact commands). Jaeger 1.62 in
particular is missing from some mirrors — 1.60 is widely mirrored.

### 13. SQLite `:memory:` is per-connection

Each connection to `file::memory:` gets its own private database. With a
connection pool, your migration lands on connection A, the next query runs on
connection B which sees an empty DB. Fix: use `file::memory:?cache=shared`
**and** set `DB_MAX_OPEN_CONNS=1`. The `internal/db/db_test.go` test does
exactly this.

### 14. Pass `*gorm.DB` via the handler constructor

Don't reach for a package-level global `db.DB`. When you start using Gorm in
business logic, change `handler.New()` to `handler.New(gdb)` and store it on
the struct. Keeps tests able to swap a sqlite memory DB.

### 15. Missing `config.yaml` is **not** an error

`config.Load` falls back to built-in defaults when the file is absent — that
way tests and scratch runs work without authoring a config. If your prod
deployment requires the file (e.g. you don't want to silently boot with
defaults), check existence yourself before calling `Load`, or fail validation
in `internal/config.validate(...)` for envs where the defaults are unsafe.

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

The template has no LICENSE file. Add one (MIT, Apache-2.0, etc.) before publishing.
