# oas-go-template

A Go project template based on OpenAPI Specification (OAS) 3.x, using
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate
server stubs and client SDK from a single source of truth: `spec/openapi.yaml`.

## Tech Stack

- Go 1.23+
- gin (HTTP framework)
- oapi-codegen (code generation)
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

## Local Observability Stack

`docker-compose.yml` boots an OpenTelemetry Collector + Jaeger all-in-one so you
can verify traces end-to-end without any cloud account.

```bash
make dev-stack                                       # start collector + Jaeger
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
  ./bin/server                                       # point the server at the collector
# in another shell, generate some traffic:
curl -sf http://localhost:8080/healthz
curl -sf http://localhost:8080/version
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

## Workflow

1. Edit `spec/openapi.yaml`.
2. Run `make gen`.
3. Implement business logic in `internal/handler/`.
4. Run `make build && ./bin/server`.

## License

TBD
