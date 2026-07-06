# oas-go-template

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go project template where **`spec/openapi.yaml` is the single source of truth**.
Server stubs and the client SDK are generated from it via
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) (StrictServerInterface
mode). All other code (config, otel, logging, db, handlers, errcode) supports
that contract.

Ships with: gin + strict-server codegen, gorm (opt-in), OTel traces+metrics
(OTLP + Prometheus pull), slog with trace_id injection, Dockerfile, Helm chart,
golangci-lint v2 config, and a Vite + React + TS frontend (deployed separately).

## Table of Contents

- [Tech Stack](#tech-stack)
- [Initialize a New Project from This Template](#initialize-a-new-project-from-this-template)
- [Quickstart](#quickstart)
- [Configuration](#configuration)
- [Database (Gorm)](#database-gorm)
- [Local Observability Stack](#local-observability-stack)
- [Daily Workflow](#daily-workflow)
- [License](#license)

## Tech Stack

- Go 1.25+
- gin (HTTP framework)
- oapi-codegen v2 (code generation, StrictServerInterface mode)
- Gorm (ORM, postgres/mysql/sqlite — opt-in)
- OpenTelemetry (traces via OTLP HTTP, metrics via OTLP + Prometheus pull)
- slog (structured logging, trace_id injected per request)
- React + Vite + TypeScript (frontend, independent deploy)
- Docker / golangci-lint v2 / Make / Helm

## Initialize a New Project from This Template

The repo ships `scripts/init-project.sh`, a one-shot renamer that rewrites the
module path and project name everywhere they appear (Go imports, Makefile,
Dockerfile, golangci-lint config, Helm chart, README/CLAUDE/CONTRIBUTING
titles). It also re-runs `make gen` so the generated code matches the new
package.

```bash
# 1. Copy the template to where your new project lives
cp -r /path/to/oas-go-template ./my-project
cd my-project
rm -rf .git bin client && git init && git branch -m main

# 2. Run the renamer with your module path
./scripts/init-project.sh github.com/yourorg/my-project

# 3. (Manual) Set chart image repos + author/copyright — script prints what's left
#    Open chart/values.yaml and edit server.image.repository / web.image.repository.
#    Edit README.md (© line) and chart/Chart.yaml (maintainers).

# 4. Verify
make build test lint
```

The script derives the OLD module path from `go.mod` — no hard-coded
`"oas-go-template"` string — so it's safe to re-run and won't match itself.
Pass a second arg to override the short name (defaults to the last segment of
the module path):

```bash
./scripts/init-project.sh github.com/yourorg/monorepo my-service
```

For the full walkthrough — what the script touches, what it skips, manual
follow-ups, and a deep dive on every configuration trap that bit the original
build — see **[SKILL.md](SKILL.md)**.

## Quickstart

For a project you've already initialized (or to explore the template itself):

```bash
make gen       # regenerate *.gen.go from spec/openapi.yaml
make build     # build cmd/server and cmd/client into bin/
make run       # go run cmd/server with version ldflags
make test      # go test -race -cover ./...
make lint      # golangci-lint v2 (excludes *.gen.go, forbids legacy log)
make audit     # govulncheck + gosec (CI gate; non-zero on any finding)
make docker    # build server image (pass GOPROXY=... if behind GFW)
```

## Configuration

All runtime config lives in `config.yaml` (YAML only — no env overlay). Copy
the example to start:

```bash
cp config.example.yaml config.yaml
./bin/server                       # picks up ./config.yaml automatically
./bin/server -c /etc/app/prod.yaml # or pass an explicit path
```

`config.yaml` is gitignored — only `config.example.yaml` is tracked. Secrets
(DSN, OTLP endpoint, etc.) live in your local `config.yaml`, never in git.

Missing `config.yaml` is fine — built-in defaults take over so tests and
scratch runs don't need to author one. Validation (`gin_mode`, `log.format`,
`db.driver` whitelist, etc.) runs after YAML has been merged into defaults.

## Database (Gorm)

DB is **opt-in**. Set `db.driver` in `config.yaml` and the server connects at
boot; leave it empty and the server runs DB-free (`/readyz` reports 503 in
that case — graceful degradation, not panic).

```yaml
# config.yaml
db:
  driver: postgres                              # postgres | mysql | sqlite; empty = disabled
  dsn: "host=localhost user=app password=app dbname=app sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  log_sql: false                                # flip to true to log every SQL statement
```

| yaml | default | notes |
|------|---------|-------|
| `db.driver` | empty | `postgres` / `mysql` / `sqlite`; empty = disabled |
| `db.dsn` | — | required when driver is set |
| `db.max_open_conns` | `25` | |
| `db.max_idle_conns` | `5` | |
| `db.conn_max_lifetime` | `30m` | any `time.ParseDuration` form |
| `db.log_sql` | `false` | `true` routes every SQL statement through gorm's Trace |

Every SQL operation becomes an OTel span via `gorm.io/plugin/opentelemetry`.
For sqlite tests use `file::memory:?cache=shared` plus `max_open_conns: 1`
(see `internal/db/db_test.go`) — without that, each pool connection gets its
own private memory DB.

## Local Observability Stack

`docker-compose.yml` boots an OpenTelemetry Collector + Jaeger all-in-one so you
can verify traces end-to-end without any cloud account.

```bash
make dev-stack                                       # start collector + Jaeger
./bin/server                                         # reads config.yaml (otel.exporter_otlp_endpoint → collector)
# in another shell, generate some traffic:
curl -sf http://localhost:8000/healthz
curl -sf http://localhost:8000/version
# open Jaeger UI:
open http://localhost:16686                          # search Service = <serviceName>
make dev-stack-down                                  # stop when done
```

Each log line carries `trace_id` / `span_id` because `otelgin.Middleware`
runs before `logging.Middleware` (see `cmd/server/main.go`). Paste the
`trace_id` straight into Jaeger's "Find a trace" box to jump from a log
entry to the corresponding trace.

`GET /metrics` serves Prometheus format off `prometheus.DefaultRegisterer`
(Go runtime + process collectors always present; OTel-translated app metrics
added when OTel is enabled). The endpoint is intentionally not in
`spec/openapi.yaml` — it's an ops endpoint, not part of the API contract.

If `docker compose up` can't pull images, configure a Docker registry
mirror in your daemon, or pull images from a CN-friendly mirror and re-tag:

```bash
docker pull docker.1ms.run/jaegertracing/all-in-one:1.60
docker pull docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0
docker tag docker.1ms.run/jaegertracing/all-in-one:1.60 jaegertracing/all-in-one:1.60
docker tag docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0 otel/opentelemetry-collector-contrib:0.110.0
```

## Daily Workflow

Once initialized, the dev loop is:

1. Edit `spec/openapi.yaml`.
2. Run `make gen` → regenerates `internal/api/*.gen.go` and `pkg/api/*.gen.go`.
3. Implement business logic in `internal/handler/` — methods return typed
   `ResponseObject` values (`api.GetFoo200JSONResponse`, etc.).
4. Run `make build && ./bin/server`.

If a handler method is missing, the compile-time assertion
`var _ api.StrictServerInterface = (*Handler)(nil)` in
`internal/handler/handler_test.go` fails the build with a clear error listing
every missing method.

## License

[MIT](LICENSE) © piwriw

> Derived projects: replace this line with your own copyright — the
> `init-project.sh` script flags it as a manual follow-up since it can't
> infer the new holder.
