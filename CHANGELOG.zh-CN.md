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
- 可配置的 CORS 策略，支持预检请求、凭据、暴露响应头，并为被拒绝的来源返回稳定的类型化 403 响应。
- 可扩展的全局 Gin middleware 链，并提供明确的自定义 middleware 扩展入口。
- `make tools` 命令，用于安装固定版本的开发工具。
- 面向业务接口的 URL 前缀 API 版本化策略；OpenAPI 契约明确列出保持不带版本的运维探针。
- 弃用接口的 OpenAPI 元数据校验，以及运行时 `Deprecation` / `Sunset` 响应头。
- 固定 `oasdiff` 版本的 `make contract-check` 契约兼容性检查，并在 pull request 中与目标分支契约比较。
- 在 `go.mod`、本地检查、CI 和后端 builder 镜像之间统一使用 Go 1.26.5。
- 使用明确版本的 Docker 基础镜像 tag，并固定 GitHub Actions 的不可变引用；新增 `make supply-chain-check` 检测漂移。
- Kubernetes 优雅摘流：服务关闭时先让 readiness 返回 503，等待配置的摘流窗口后再关闭监听器。

### 变更

- CI 改为使用固定版本的代码生成、lint、安全扫描和 Helm 工具，不再依赖浮动的 `latest` 版本。
- Helm 默认在没有 collector 时关闭 OTel，并启用更安全的 non-root Pod 安全默认值。
- 结构化错误日志：内部错误详情和 panic 堆栈只记录在日志中，不返回给外部调用方。
- CI 不再执行可变 tag 的 Helm 安装脚本，改为使用 SHA 固定的 `setup-helm` action。
- 服务启动时会拒绝不符合版本化规则的路径，以及缺少或包含无效下线日期的弃用接口。
- Helm 部署新增 `terminationGracePeriodSeconds`，为摘流窗口预留时间。

### 修复

- 出站请求耗尽最后一次重试后不再继续等待，也不会在调用方读取前消费最终错误响应体。
- HTTP 监听启动失败现在会稳定返回原始错误，不再与优雅关闭的取消信号竞争并被误判为正常退出。
- readiness 响应不再暴露数据库驱动的原始错误，详细诊断信息只保留在服务端日志中。
- 升级 PostgreSQL 和 ClickHouse 相关传递依赖，修复固定安全审计发现的漏洞。

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
