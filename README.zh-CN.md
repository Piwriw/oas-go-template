# oas-go-template

[![CI](https://github.com/piwriw/oas-go-template/actions/workflows/ci.yml/badge.svg)](https://github.com/piwriw/oas-go-template/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/piwriw/oas-go-template)](https://goreportcard.com/report/github.com/piwriw/oas-go-template)
[![Go Reference](https://pkg.go.dev/badge/github.com/piwriw/oas-go-template.svg)](https://pkg.go.dev/github.com/piwriw/oas-go-template)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**[English](README.md)** | 简体中文

一个 Go 项目模板，**以 `spec/openapi.yaml` 为唯一事实来源**。服务端 stub 和客户端 SDK 通过 [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)（StrictServerInterface 模式）从 OAS 自动生成。所有其它代码（config、otel、logging、db、handler、errcode）都是为这份契约服务的辅助层。

开箱即用：gin + strict-server 代码生成、Gorm（可选）、OpenTelemetry traces+metrics（OTLP 推送 + Prometheus 拉取）、带 trace_id 注入的 slog、Dockerfile、Helm chart、golangci-lint v2 配置，以及独立部署的 Vite + React + TS 前端。

## 目录

- [技术栈](#技术栈)
- [从模板初始化新项目](#从模板初始化新项目)
- [快速开始](#快速开始)
- [配置](#配置)
- [数据库 (Gorm)](#数据库-gorm)
- [本地可观测性栈](#本地可观测性栈)
- [日常开发流程](#日常开发流程)
- [API 契约](#api-契约)
- [变更日志](CHANGELOG.zh-CN.md)
- [License](#license)

## 技术栈

- Go 1.25+
- gin（HTTP 框架）
- oapi-codegen v2（代码生成，StrictServerInterface 模式）
- Gorm（ORM，支持 postgres/mysql/sqlite — 可选启用）
- OpenTelemetry（traces 走 OTLP HTTP，metrics 走 OTLP + Prometheus 拉取）
- slog（结构化日志，按请求注入 trace_id）
- React + Vite + TypeScript（前端，独立部署）
- Docker / golangci-lint v2 / Make / Helm

## 从模板初始化新项目

仓库自带 `scripts/init-project.sh`——一键重命名脚本。最快的路径是把下面的流程交给 AI 编码代理（Claude Code、Cursor、Cline 等），它会驱动脚本、追问无法推断的值、并验证结果。

复制下面的 prompt，填好变量（或留空让 AI 来问），粘贴到你的 AI 工具里：

````markdown
帮我从 oas-go-template 派生一个新的 Go 项目。

输入参数（缺哪个就先问我哪个，问完再开始执行）：
- TARGET_PATH   : 新项目要放的位置
- MODULE_PATH   : 例如 github.com/yourorg/my-project
- SHORT_NAME    : 可选；默认取 MODULE_PATH 的最后一段
- GITHUB_HOSTED : yes / no —— 新项目是否托管在 GitHub.com？
                  no  → 删除 .github/（CI workflow、Dependabot、issue 模板、
                         security advisory URL 全都依赖 GitHub）。

执行步骤：
1. git clone https://github.com/piwriw/oas-go-template.git "$TARGET_PATH"
2. cd "$TARGET_PATH"
3. rm -rf .git bin client && git init -q && git branch -m main
4. ./scripts/init-project.sh "$MODULE_PATH" "$SHORT_NAME"
5. 如果 GITHUB_HOSTED != yes：rm -rf .github/
   （否则保留。重命名脚本已经把 .github/ 里所有 github.com/piwriw/oas-go-template
    的 URL 改写成新的模块路径，CI / Dependabot / issue 模板都能继续工作。）
6. 脚本会输出一段 "Manual follow-ups"。逐条处理：
   a. chart/values.yaml —— 问我镜像仓库地址，更新 server.image.repository
      和 web.image.repository。
   b. README.md 的 © 行、chart/Chart.yaml 的 maintainers —— 问我作者署名，
      替换掉 piwriw。
7. 按顺序验证：
   - golangci-lint config verify    # 应无任何输出
   - make gen                       # 应无 diff
   - make build test lint           # 全部绿
8. 用一段话汇报：改了什么、还剩哪些事让我自己做（比如"编辑 spec/openapi.yaml
   定义你的 API，再跑一次 make gen"）。

执行前请阅读 SKILL.md，了解重命名脚本改了哪些位置、跳过了哪些、以及所有
要避开的配置陷阱。第 6 步之后、第 7 步之前，必须等我确认 registry 和
author 的值再继续。
````

想手动执行？看 `./scripts/init-project.sh --help` 和 **[SKILL.md](SKILL.md)** 里的底层命令与完整流程。

## 快速开始

针对已经初始化的项目（或仅想体验模板本身）：

```bash
make gen       # 从 spec/openapi.yaml 重新生成 *.gen.go（固定 oapi-codegen v2.7.1）
make tools     # 安装固定版本的开发工具
make build     # 编译 cmd/server 和 cmd/client 到 bin/
make run       # 带版本 ldflags 的 go run cmd/server
make test      # go test -race -cover ./...
make lint      # golangci-lint v2（排除 *.gen.go，禁止 legacy log 包）
make audit     # govulncheck v1.6.0 + gosec v2.27.1（CI 门禁）
make docker    # 构建服务端镜像（在 GFW 后请传 GOPROXY=...）
```

## 配置

所有运行时配置都在 `config.yaml`（仅 YAML——没有环境变量覆盖层）。复制示例开始：

```bash
cp config.example.yaml config.yaml
./bin/server                       # 自动读取 ./config.yaml
./bin/server -c /etc/app/prod.yaml # 或显式传路径
```

`config.yaml` 已加入 `.gitignore`——仓库只追踪 `config.example.yaml`。密钥（DSN、OTLP endpoint 等）放在你本地的 `config.yaml` 里，绝不入库。

`config.yaml` 缺失也没关系——内置默认值会接管，测试和临时跑跑不用准备配置文件。校验（`gin_mode`、`log.format`、`db.driver` 白名单等）在 YAML 合并到默认值之后执行。

在 Kubernetes 中，可设置 Helm 的 `server.existingConfigSecret.name`，挂载包含完整 `config.yaml` 的 Secret。这样既保持仅 YAML 的配置模型，又能避免 DSN、exporter 凭据等密钥进入 chart values。

HTTP 服务默认使用 5 秒读请求头超时、15 秒读超时、30 秒写超时、60 秒空闲超时，限制请求头为 1 MiB、请求体为 1 MiB。可在 `config.yaml` 的 `server` 下调整；流式响应可将 `write_timeout: 0`，应用层请求体限制可将 `max_body_bytes: 0` 关闭。

## 数据库 (Gorm)

数据库是**可选启用的**。在 `config.yaml` 设置 `db.driver`，服务启动时连接；留空则不启用数据库。禁用的数据库不是已配置依赖，因此此模式下 `/readyz` 仍返回 200。

```yaml
# config.yaml
db:
  driver: postgres                              # postgres | mysql | sqlite；空 = 禁用
  dsn: "host=localhost user=app password=app dbname=app sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  log_sql: false                                # 改成 true 把每条 SQL 都打到日志
```

| yaml | 默认值 | 说明 |
|------|---------|-------|
| `db.driver` | 空 | `postgres` / `mysql` / `sqlite`；空 = 禁用 |
| `db.dsn` | — | 启用 driver 时必填 |
| `db.max_open_conns` | `25` | |
| `db.max_idle_conns` | `5` | |
| `db.conn_max_lifetime` | `30m` | 任意 `time.ParseDuration` 接受的形式 |
| `db.log_sql` | `false` | `true` 时把每条 SQL 经 gorm 的 Trace 输出 |

每条 SQL 操作都会通过 `gorm.io/plugin/opentelemetry` 成为 OTel span。sqlite 测试用 `file::memory:?cache=shared` 加 `max_open_conns: 1`（见 `internal/db/db_test.go`）——否则连接池里每个连接会拿到独立的内存数据库。

## 本地可观测性栈

`docker-compose.yml` 启动一个 OpenTelemetry Collector + Jaeger all-in-one，让你无需任何云账号就能端到端验证 trace。

```bash
make dev-stack                                       # 启动 collector + Jaeger
./bin/server                                         # 读 config.yaml（otel.exporter_otlp_endpoint → collector）
# 另开一个 shell 制造流量：
curl -sf http://localhost:8000/healthz
curl -sf http://localhost:8000/version
# 打开 Jaeger UI：
open http://localhost:16686                          # Service = <serviceName>
make dev-stack-down                                  # 用完关掉
```

每条日志都带 `trace_id` / `span_id`，因为 `otelgin.Middleware` 跑在 `logging.Middleware` **之前**（见 `cmd/server/main.go`）。把 `trace_id` 直接粘到 Jaeger 的 "Find a trace" 框里，就能从日志跳到对应的 trace。

`GET /metrics` 从 `prometheus.DefaultRegisterer` 输出 Prometheus 格式（Go runtime + process 指标始终存在；启用 OTel 后还会加上 OTel 翻译过来的应用指标）。这个端点**故意不在** `spec/openapi.yaml` 里——它是运维端点，不属于 API 契约。

如果 `docker compose up` 拉不动镜像，在 Docker daemon 里配 registry mirror，或从国内镜像拉取后重打 tag：

```bash
docker pull docker.1ms.run/jaegertracing/all-in-one:1.60
docker pull docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0
docker tag docker.1ms.run/jaegertracing/all-in-one:1.60 jaegertracing/all-in-one:1.60
docker tag docker.1ms.run/otel/opentelemetry-collector-contrib:0.110.0 otel/opentelemetry-collector-contrib:0.110.0
```

## 日常开发流程

项目初始化后，开发循环是：

1. 编辑 `spec/openapi.yaml`。
2. 跑 `make gen` → 重新生成 `internal/api/*.gen.go` 和 `pkg/api/*.gen.go`。
3. 在 `internal/handler/` 实现业务逻辑——方法返回有类型的 `ResponseObject`（如 `api.GetFoo200JSONResponse`）。
4. 跑 `make build && ./bin/server`。

如果漏写了某个 handler 方法，`internal/handler/handler_test.go` 里的编译期断言 `var _ api.StrictServerInterface = (*Handler)(nil)` 会让构建失败，并列出所有缺失的方法。

## API 契约

API 使用 URL 前缀版本化。由于 Kubernetes 和负载均衡器依赖稳定地址，运维探针 `/healthz`、`/readyz` 和 `/version` 保持不带版本；后续业务接口必须放在 `/vN/` 下（例如 `/v1/orders`）。这项规则由 `spec/openapi.yaml` 中的 `x-api-version` 和 `x-versioning` 声明，并在服务启动时校验。

弃用接口时设置 `deprecated: true`，并提供 RFC3339 格式的两个日期扩展：

```yaml
deprecated: true
x-deprecation-date: "2026-08-01T00:00:00Z"
x-sunset-date: "2027-02-01T00:00:00Z"
```

服务会校验下线日期晚于弃用日期，并在响应中加入对应的 `Deprecation` 和 `Sunset` 响应头。接口应持续可用到下线日期；提前删除会被视为破坏性变更。
启用 CORS 时，如果浏览器客户端需要读取这些响应头，请将它们加入 `cors.expose_headers`。

本地可以把当前契约与基线文件比较：

```bash
make contract-check BASE_SPEC=/path/to/openapi-base.yaml
```

Pull request 会自动使用目标分支的 spec，并通过固定版本的 `oasdiff` v1.10.28 执行同一检查。确实需要破坏性变更时，应创建新的 `/vN` API 版本并提供明确的迁移方案。

## License

[MIT](LICENSE) © piwriw

> 派生项目：请把这行替换为你自己的版权声明——`init-project.sh` 脚本会把这个标为手动 follow-up，因为它无法推断新的持有者。
