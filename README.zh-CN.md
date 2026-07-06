# oas-go-template

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

仓库自带 `scripts/init-project.sh`——一个一键重命名脚本，会把模块路径和项目名同步替换到所有出现位置（Go import、Makefile、Dockerfile、golangci-lint 配置、Helm chart、README/CLAUDE/CONTRIBUTING 标题），并自动跑一次 `make gen` 让生成代码与新包名匹配。

```bash
# 1. 把模板复制到新项目目录
cp -r /path/to/oas-go-template ./my-project
cd my-project
rm -rf .git bin client && git init && git branch -m main

# 2. 用你的模块路径跑重命名脚本
./scripts/init-project.sh github.com/yourorg/my-project

# 3. （手动）设置 chart 的 image repos 和作者署名 —— 脚本会提示哪些没自动改
#    打开 chart/values.yaml，编辑 server.image.repository / web.image.repository。
#    编辑 README.md（© 行）和 chart/Chart.yaml（maintainers）。

# 4. 验证
make build test lint
```

脚本从 `go.mod` 推导旧模块路径——**不写死 `"oas-go-template"`**——所以可以安全重复执行，也不会误伤自己。需要覆盖短名时传第二个参数（默认取模块路径的最后一段）：

```bash
./scripts/init-project.sh github.com/yourorg/monorepo my-service
```

完整流程——脚本改了什么、跳过了什么、需要手动补什么、以及原版构建中踩过的每一个配置陷阱，请看 **[SKILL.md](SKILL.md)**。

## 快速开始

针对已经初始化的项目（或仅想体验模板本身）：

```bash
make gen       # 从 spec/openapi.yaml 重新生成 *.gen.go
make build     # 编译 cmd/server 和 cmd/client 到 bin/
make run       # 带版本 ldflags 的 go run cmd/server
make test      # go test -race -cover ./...
make lint      # golangci-lint v2（排除 *.gen.go，禁止 legacy log 包）
make audit     # govulncheck + gosec（CI 门禁；任一发现即非零退出）
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

## 数据库 (Gorm)

数据库是**可选启用的**。在 `config.yaml` 设置 `db.driver`，服务启动时连接；留空则不启用数据库（`/readyz` 返回 503 表示未就绪——优雅降级，不 panic）。

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

## License

[MIT](LICENSE) © piwriw

> 派生项目：请把这行替换为你自己的版权声明——`init-project.sh` 脚本会把这个标为手动 follow-up，因为它无法推断新的持有者。
