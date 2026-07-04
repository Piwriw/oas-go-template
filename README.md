# oas-go-template

A Go project template based on OpenAPI Specification (OAS) 3.x, using
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate
server stubs and client SDK from a single source of truth: `spec/openapi.yaml`.

## Tech Stack

- Go 1.23+
- gin (HTTP framework)
- oapi-codegen (code generation)
- Gorm (ORM, postgres/mysql/sqlite)
- React + Vite + TypeScript (frontend, deployed separately)
- Docker / golangci-lint / Make

## Project Layout

See `docs/superpowers/specs/2026-07-04-project-init-design.md` (removed from tracking in this template; see git history if needed) and `SKILL.md` for how to derive a new project from this one.

## Quickstart

```bash
make gen       # generate code from spec/openapi.yaml
make build     # build cmd/server and cmd/client
make test      # run tests
make lint      # run golangci-lint
make docker    # build server docker image
```

## Configuration

All runtime config lives in `config.yaml`. Copy the example to start:

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
open http://localhost:16686                          # search Service = oas-go-template
make dev-stack-down                                  # stop when done
```

Each log line carries `trace_id` / `span_id` because `otelgin.Middleware`
runs before `logging.Middleware` (see `cmd/server/main.go`). Paste the
`trace_id` straight into Jaeger's "Find a trace" box to jump from a log
entry to the corresponding trace.

If `docker compose up` can't pull images, configure a Docker registry
mirror in your daemon, or pull images from a CN-friendly mirror and re-tag:

```bash
docker pull docker.1ms.run/jaegertracing/all-in-one:1.60
docker pull docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0
docker tag docker.1ms.run/jaegertracing/all-in-one:1.60 jaegertracing/all-in-one:1.60
docker tag docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0 otel/opentelemetry-collector-contrib:0.110.0
```

## Database (Gorm)

DB is **opt-in**. Set `db.driver` in `config.yaml` and the server connects at
boot; leave it empty and the server runs DB-free.

```yaml
# config.yaml
db:
  driver: postgres                              # postgres | mysql | sqlite; empty = disabled
  dsn: "host=localhost user=app password=app dbname=app sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
```

| yaml | default | notes |
|------|---------|-------|
| `db.driver` | empty | `postgres` / `mysql` / `sqlite`; empty = disabled |
| `db.dsn` | — | required when driver is set |
| `db.max_open_conns` | `25` | |
| `db.max_idle_conns` | `5` | |
| `db.conn_max_lifetime` | `30m` | any `time.ParseDuration` form |

Every SQL operation becomes an OTel span via `gorm.io/plugin/opentelemetry`.
For sqlite tests use `file::memory:?cache=shared` plus `max_open_conns: 1`
(see `internal/db/db_test.go`).

## Workflow

1. Edit `spec/openapi.yaml`.
2. Run `make gen`.
3. Implement business logic in `internal/handler/`.
4. Run `make build && ./bin/server`.

## License

TBD
