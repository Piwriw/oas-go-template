# Changelog

All notable changes to this project are documented here.

English | **[简体中文](CHANGELOG.zh-CN.md)**

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Runtime request validation against the embedded OpenAPI contract.
- A consistent `api.Error` response for request parsing, routing, handler,
  response serialization, and panic failures.
- Configurable HTTP read, write, idle, header, and request-body limits.
- Typed client response fields for standard 400, 404, 405, 413, and 500 API
  errors.
- Structured error logging that keeps internal details and panic stack traces
  out of public responses.

## [0.1.0] - 2026-07-19

Initial template baseline.

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

### Changed

- The default backend port is `8000`.
- Runtime configuration is YAML-only, with built-in defaults and validation.

### Fixed

- DB-free deployments now report Ready when no database dependency is
  configured; configured database failures still make readiness fail.
