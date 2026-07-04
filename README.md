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

## Workflow

1. Edit `spec/openapi.yaml`.
2. Run `make gen`.
3. Implement business logic in `internal/handler/`.
4. Run `make build && ./bin/server`.

## License

TBD
