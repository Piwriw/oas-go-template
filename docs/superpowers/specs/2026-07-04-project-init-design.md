# oas-go-template 项目初始化设计

**日期**: 2026-07-04
**状态**: 已批准待实施
**作者**: piwriw

---

## 1. 项目概述

`oas-go-template` 是一个基于 OpenAPI Specification (OAS) 3.x 的 Go 项目模板。它以 OAS spec 文件作为唯一契约源(single source of truth),通过 [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) 自动生成 server stub 和 client SDK 代码,从而保证契约、server 实现、client SDK 三者始终一致。

模板目标用户:需要构建 RESTful API 服务并同时为外部消费方提供 SDK 的 Go 开发者。

---

## 2. 技术栈

| 维度 | 选型 | 版本/说明 |
|------|------|----------|
| 语言 | Go | 1.23+ |
| HTTP 框架 | gin | server 端路由与中间件 |
| 代码生成 | oapi-codegen | 生成类型、server stub、client SDK |
| 前端 | React + Vite + TypeScript | 独立部署,不内嵌 |
| 部署 | 前后端分离 | server 跑在 Docker 容器,前端走 CDN/Nginx |
| Go module | `github.com/piwriw/oas-go-template` | |

---

## 3. 目录结构

```
oas-go-template/
├── spec/
│   └── openapi.yaml          # 源 OAS spec(项目核心契约)
├── cmd/
│   ├── server/
│   │   └── main.go           # server 入口
│   └── client/
│       └── main.go           # client 示例调用入口
├── internal/
│   ├── api/                  # server 端生成代码(package api)
│   │   ├── types.gen.go      # 共享类型(供 server/handler 使用)
│   │   └── server.gen.go     # gin server 接口与绑定
│   ├── handler/              # 业务实现(实现 internal/api.StrictServerInterface)
│   ├── middleware/           # 自定义中间件
│   └── config/               # 配置加载
├── pkg/
│   └── api/                  # 对外发布的 client SDK(package api)
│       ├── types.gen.go      # client 用类型(独立一份)
│       └── client.gen.go     # client SDK 实现
├── web/                      # 前端项目(React + Vite + TypeScript)
│   ├── src/
│   ├── public/
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── README.md
├── build/
│   └── Dockerfile            # 后端镜像构建(server only)
├── scripts/
│   └── gen.sh                # 代码生成脚本(可选,Makefile 也能调用 oapi-codegen)
├── docs/
│   └── superpowers/specs/    # 设计文档
├── .gitignore
├── .golangci.yml             # Go 静态检查配置
├── .editorconfig             # 编辑器统一配置(可选)
├── oapi-codegen.yaml         # oapi-codegen 配置(server/client 分段输出)
├── go.mod
├── go.sum
├── Makefile                  # make gen / build / test / lint / docker
└── README.md
```

---

## 4. 模块职责

### 4.1 `spec/`
唯一契约源。OAS 3.x YAML 文件,所有 server 接口、类型、client SDK 都从此处生成。
修改契约必须先修改 `spec/openapi.yaml`,然后运行 `make gen`。

### 4.2 `cmd/`
二进制入口,每个子目录对应一个可执行程序,只做组装不做业务。

- `cmd/server/main.go`:加载配置、注册中间件、把 `handler` 实现挂到生成的 gin 路由上、启动 HTTP 服务
- `cmd/client/main.go`:示例程序,演示如何用 `pkg/api` 的 client 调用远端 API

### 4.3 `internal/`(项目内部,不可外部 import)

- **`internal/api/`**:oapi-codegen 生成的 server 端代码
  - `types.gen.go`:OAS schema 对应的 Go 类型,server/handler 共用
  - `server.gen.go`:gin 路由绑定、请求参数解析、`StrictServerInterface` 接口定义
- **`internal/handler/`**:实现 `internal/api.StrictServerInterface`,写业务逻辑
- **`internal/middleware/`**:gin 中间件(认证、日志、recovery 等)
- **`internal/config/`**:配置加载(从环境变量 / yaml / flags)

### 4.4 `pkg/`(对外暴露,可被外部项目 import)

- **`pkg/api/`**:oapi-codegen 生成的 client SDK
  - `types.gen.go`:client 用类型(与 `internal/api/types.gen.go` 独立)
  - `client.gen.go`:HTTP client 实现,带请求/响应类型安全绑定

### 4.5 `web/`
React + Vite + TypeScript 前端项目,独立部署。
模板预置基础结构(`App.tsx`、`main.tsx`、Vite 配置),不预置业务页面。
通过 `pkg/api` 的 TypeScript 等价物(后续可用 openapi-typescript 或 openapi-fetch 生成)调用后端。

### 4.6 `build/`
- **`build/Dockerfile`**:多阶段构建后端镜像
  - Stage 1:Go 构建环境,编译 `cmd/server` 为静态二进制
  - Stage 2:`alpine` 或 `distroless` 运行环境
- 不构建前端(前后端分离)

### 4.7 `scripts/`
- **`scripts/gen.sh`**:封装 `oapi-codegen` 命令,被 Makefile 调用

### 4.8 根目录配置文件

| 文件 | 作用 |
|------|------|
| `oapi-codegen.yaml` | oapi-codegen 配置,分 server-types / server / client 三段输出 |
| `.golangci.yml` | golangci-lint 规则(可关闭对 `*.gen.go` 的检查) |
| `.editorconfig` | 编辑器统一缩进、行尾、文件结尾换行 |
| `.gitignore` | 忽略 `/dist`、`/bin`、`node_modules`、生成产物(可选) |
| `Makefile` | 顶层命令:`gen` / `build` / `test` / `lint` / `docker` / `dev` |
| `go.mod` / `go.sum` | Go modules |

---

## 5. 代码生成工作流

### 5.1 oapi-codegen 配置(`oapi-codegen.yaml`)

模板采用分段生成策略,3 段输出对应 3 个文件:

```yaml
# 示例配置结构,具体字段以 oapi-codegen 版本为准
package: api
generate:
  - models
  - gin-server
  - strict-server
output: internal/api/server.gen.go
output-options:
  skip-prune: false
```

实际通过 3 次调用或单一配置的多段输出,分别生成:

| 输出文件 | package | 内容 |
|---------|---------|------|
| `internal/api/types.gen.go` | `api` | models(types) |
| `internal/api/server.gen.go` | `api` | gin-server + strict-server |
| `pkg/api/client.gen.go` + `pkg/api/types.gen.go` | `api` | client + 自带 types |

> **types 双份策略**:由于 `internal/` 不能被 `pkg/` 导入,client 端必须自带 types。这是 Go 模块的硬性约束。如果未来发现 types 维护成本高,可考虑把共享 types 提到 `pkg/api/types` 包,但当前默认走双份。

### 5.2 Makefile 目标

```makefile
.PHONY: gen build test lint docker dev

gen:        ## 从 spec/openapi.yaml 生成代码
	./scripts/gen.sh

build:      ## 构建 cmd/server 和 cmd/client
	go build -o bin/server ./cmd/server
	go build -o bin/client ./cmd/client

test:       ## 跑测试
	go test ./...

lint:       ## golangci-lint
	golangci-lint run

docker:     ## 构建后端镜像
	docker build -f build/Dockerfile -t oas-go-template:latest .

dev:        ## 启动后端热重载(air)
	air
```

---

## 6. 前端 `web/` 子结构(预置基线)

```
web/
├── src/
│   ├── App.tsx
│   ├── main.tsx              # React 入口
│   └── api/                  # (预留)前端用的 OAS client 生成产物
├── public/
├── index.html
├── package.json              # react, react-dom, vite, typescript
├── tsconfig.json
├── vite.config.ts
├── .eslintrc.cjs             # 可选
├── .gitignore                # node_modules, dist 等
└── README.md
```

前端 OAS client 不在本模板初始化范围内,预留 `src/api/` 目录占位。

---

## 7. 部署模型

```
+--------------------+        +--------------------+
|   Browser          |        |   API Consumer     |
|   (React SPA)      |        |   (uses pkg/api)   |
+---------+----------+        +---------+----------+
          |                             |
          |  HTTPS                      |  HTTPS
          v                             v
+---------+----------+        +---------+----------+
|  CDN / Nginx       |        |  Go Server        |
|  (前端静态资源)    |        |  (cmd/server)     |
+--------------------+        |  in Docker        |
                              +--------------------+
```

- 前端构建产物 `web/dist/` 由 CDN/Nginx 提供
- 后端只跑 `cmd/server`,CORS 在 server 端中间件层处理
- `pkg/api` 通过 Go module 被消费方项目引用

---

## 8. 范围内 vs 范围外

### 范围内(本次初始化包含)

- 上述完整目录结构
- `go.mod`、空 main.go 占位、根 `.gitignore`、`.golangci.yml`(基础配置)
- `Makefile`、`scripts/gen.sh`(空脚本占位)
- `oapi-codegen.yaml`(配置骨架)
- `build/Dockerfile`(多阶段构建骨架)
- `web/` 基础 Vite + React + TS 项目(`npm create vite@latest` 默认输出)
- 各目录的 `README.md` 占位(可选)

### 范围外(本次不包含,留待后续)

- OAS spec 的具体业务字段(只放空骨架)
- handler/middleware/config 的具体业务实现(只占位接口)
- CI 流水线(`.github/workflows/`)
- 前端 OAS client 生成
- 单元测试样例
- 完整的 README 内容

---

## 9. 待定事项

无。所有关键决策已在 brainstorming 阶段确认。

---

## 10. 后续步骤

本设计经用户批准后,将通过 `superpowers:writing-plans` skill 生成实施计划,然后按计划逐项落地。
