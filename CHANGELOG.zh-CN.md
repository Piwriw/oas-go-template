# 变更日志

这里记录本项目的所有重要变更。

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

**[English](CHANGELOG.md)** | 简体中文

## [未发布]

### 新增

- 基于内嵌 OpenAPI 契约的运行时请求校验。
- 统一的 `api.Error` 错误响应，覆盖请求解析、路由、handler、响应序列化和 panic 场景。
- 可配置的 HTTP 读、写、空闲、请求头和请求体限制。
- 客户端可按状态码解析标准 400、404、405、413 和 500 错误响应。
- 结构化错误日志：内部错误详情和 panic 堆栈只记录在日志中，不返回给外部调用方。

## [0.1.0] - 2026-07-19

首个模板基线版本。

### 新增

- 以 `spec/openapi.yaml` 为唯一事实来源，生成严格 Gin 服务端 stub、Go 客户端 SDK 和运行时 OAS 文档。
- `/healthz`、`/readyz`、`/version` 和 Prometheus `/metrics` 端点。
- 可选的 Gorm 数据库支持，覆盖 PostgreSQL、MySQL 和 SQLite，包含连接池、启动 ping 检查及 OpenTelemetry SQL tracing。
- 通过 OTLP HTTP 和 Prometheus pull 导出 OpenTelemetry traces/metrics，并在 slog 中关联 trace、span 和 request ID。
- `pkg/httpx` 客户端工具：JSON 请求、指数退避重试、`Retry-After` 支持、W3C trace 传播和每次尝试的结构化日志。
- Docker 镜像、后端和前端 Helm chart、健康探针、可选 HPA/Ingress，以及基于 Secret 的服务端配置。
- Vite + React + TypeScript 前端脚手架，可独立通过 nginx 部署。
- 项目初始化脚本、Makefile 工作流、GitHub Actions CI、Dependabot、DCO 检查、lint 和安全审计目标。

### 变更

- 默认后端端口改为 `8000`。
- 运行时配置改为仅使用 YAML，并提供内置默认值和校验。

### 修复

- 未配置数据库依赖时，DB-free 部署现在会正确报告 Ready；已配置数据库发生故障时仍会导致 readiness 失败。
