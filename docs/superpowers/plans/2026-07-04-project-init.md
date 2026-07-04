# oas-go-template 项目初始化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 从零搭建一个基于 OAS 3.x + oapi-codegen 的 Go 项目模板,含 server、client SDK、React 前端骨架,以及 Dockerfile、Makefile、lint 配置。

**Architecture:** `spec/openapi.yaml` 是唯一契约源,通过 `oapi-codegen` 分段生成 `internal/api`(server 用)和 `pkg/api`(client SDK)代码。Server 用 gin 跑 API,前端独立部署。cmd/server 和 cmd/client 分别是两个二进制入口。

**Tech Stack:** Go 1.23+、gin、oapi-codegen、Vite + React + TypeScript、Docker、golangci-lint、Make

## Global Constraints

- **Go module path**: `github.com/piwriw/oas-go-template`(必须用这个,不可改名)
- **Go 版本下限**: 1.23(`go.mod` 的 `go` 指令为 `go 1.23`)
- **HTTP 框架**: gin(oapi-codegen 用 `gin-server` 输出模式)
- **生成代码隔离**: 任何 `*.gen.go` 文件禁止手改,只能由 `make gen` 重新生成
- **types 双份**:`internal/api/types.gen.go` 与 `pkg/api/types.gen.go` 是独立两份,因为 `pkg/` 不能 import `internal/`
- **前后端分离**: server 不内嵌前端静态文件,Dockerfile 只构建 Go 后端
- **包名约定**: server 端 `package api`(internal/api),client 端 `package api`(pkg/api)
- **commit message 前缀**: 用 `chore:` 表示脚手架/配置,`feat:` 表示功能,`docs:` 表示文档

---

## File Structure

| 路径 | 类型 | 责任 |
|------|------|------|
| `.gitignore` | 创建 | 通用 Go/Node/Docker 忽略规则 |
| `.editorconfig` | 创建 | 编辑器统一配置 |
| `.dockerignore` | 创建 | Docker build 上下文排除 |
| `.golangci.yml` | 创建 | golangci-lint 规则,排除 `*.gen.go` |
| `README.md` | 创建 | 项目说明、开发流程 |
| `Makefile` | 创建 | 顶层命令入口 |
| `go.mod` / `go.sum` | 创建 | Go modules |
| `spec/openapi.yaml` | 创建 | OAS 3.0 契约源(最小可用 spec) |
| `oapi-codegen.yaml` | 创建 | 代码生成配置 |
| `scripts/gen.sh` | 创建 | 封装 oapi-codegen 调用 |
| `cmd/server/main.go` | 创建 | server 二进制入口 |
| `cmd/client/main.go` | 创建 | client 示例入口 |
| `internal/api/*.gen.go` | 生成 | server 端类型、gin 绑定、StrictServerInterface |
| `internal/handler/` | 创建 | 实现 StrictServerInterface |
| `internal/middleware/` | 创建 | recovery / logger 等中间件 |
| `internal/config/` | 创建 | 配置加载 |
| `pkg/api/*.gen.go` | 生成 | client SDK + 自带 types |
| `web/` | 创建 | Vite + React + TS 前端 |
| `build/Dockerfile` | 创建 | 多阶段构建 server 镜像 |

---

## Task 1: Git 初始化 + 根目录基础文件

**Files:**
- Create: `.gitignore`
- Create: `.editorconfig`
- Create: `README.md`

**Interfaces:**
- Consumes: 无
- Produces: 一个 git 仓库 + 根目录基础文件,后续任务在此之上累加

- [ ] **Step 1: 初始化 git 仓库**

```bash
cd /Users/joohwan/GolandProjects/oas-go-template
git init
git branch -m main
```

Expected: `Initialized empty Git repository in ...`

- [ ] **Step 2: 写 `.gitignore`**

文件路径:`.gitignore`

```gitignore
# Binaries
/bin/
/dist/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Go test / build artifacts
*.test
*.out
coverage.txt
coverage.html

# Generated Go code (kept tracked, but in case we add anything transient)
# Note: *.gen.go files ARE tracked; do not ignore them.

# Dependency directories
/vendor/

# Editor / IDE
.idea/
.vscode/
*.swp
*.swo
.DS_Store

# Environment files
.env
.env.local
.env.*.local

# Node (web/)
node_modules/
web/dist/
web/.vite/

# Air live-reload tmp
/tmp/

# OS
Thumbs.db
```

- [ ] **Step 3: 写 `.editorconfig`**

文件路径:`.editorconfig`

```ini
root = true

[*]
end_of_line = lf
insert_final_newline = true
charset = utf-8

[*.{go,mod,sum}]
indent_style = tab
indent_size = 4

[*.{yaml,yml,json,ts,tsx,js,jsx,md}]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab
```

- [ ] **Step 4: 写 `README.md`(最小骨架)**

文件路径:`README.md`

```markdown
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

See `docs/superpowers/specs/2026-07-04-project-init-design.md`.

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
```

- [ ] **Step 5: 验证文件存在**

Run: `ls -la`
Expected: 看到 `.gitignore`、`.editorconfig`、`README.md`、`docs/`、`.idea/`

- [ ] **Step 6: 第一次 commit**

```bash
git add .gitignore .editorconfig README.md docs/
git commit -m "chore: init repo with base files and design docs"
```

Expected: 第一次 commit 创建成功

---

## Task 2: Go module + cmd 入口骨架

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `cmd/client/main.go`

**Interfaces:**
- Consumes: 无
- Produces: 一个能 `go build` 通过的空骨架,包含两个 main 函数

- [ ] **Step 1: 初始化 go module**

Run:
```bash
go mod init github.com/piwriw/oas-go-template
```

Expected: `go: creating new go.mod: ...`,文件 `go.mod` 出现,`module github.com/piwriw/oas-go-template` 字段正确

- [ ] **Step 2: 检查 `go.mod` 的 Go 版本**

Run: `head -3 go.mod`
Expected: `go 1.23` 或更高(若工具链默认写了 1.26 也接受,符合 Global Constraint "1.23+")

如果不是,手动把 `go.mod` 第三行的 `go X.Y` 改成 `go 1.23`(下限)或保留实际版本。

- [ ] **Step 3: 写 `cmd/server/main.go`(占位)**

文件路径:`cmd/server/main.go`

```go
package main

import "fmt"

func main() {
	fmt.Println("oas-go-template server: not implemented yet")
}
```

- [ ] **Step 4: 写 `cmd/client/main.go`(占位)**

文件路径:`cmd/client/main.go`

```go
package main

import "fmt"

func main() {
	fmt.Println("oas-go-template client: not implemented yet")
}
```

- [ ] **Step 5: 验证 build**

Run: `go build ./...`
Expected: 无输出(成功)

- [ ] **Step 6: 验证 vet**

Run: `go vet ./...`
Expected: 无输出(成功)

- [ ] **Step 7: 跑一下二进制确认**

Run:
```bash
go run ./cmd/server
go run ./cmd/client
```

Expected:
```
oas-go-template server: not implemented yet
oas-go-template client: not implemented yet
```

- [ ] **Step 8: Commit**

```bash
git add go.mod cmd/
git commit -m "chore: init go module and cmd entrypoints"
```

---

## Task 3: OAS spec 骨架 + oapi-codegen 配置 + 生成脚本

**Files:**
- Create: `spec/openapi.yaml`
- Create: `oapi-codegen.yaml`
- Create: `scripts/gen.sh`

**Interfaces:**
- Consumes: Task 2 的 go module
- Produces: 一份可被 oapi-codegen 处理的最小 OAS spec + 生成配置,供 Task 4 调用

- [ ] **Step 1: 写最小 OAS spec**

文件路径:`spec/openapi.yaml`

包含一个 `GET /healthz` 端点和一个 `Error` 共享 schema,作为模板的最小可用契约。

```yaml
openapi: 3.0.3
info:
  title: oas-go-template API
  version: 0.1.0
  description: |
    Template project API. Replace with your own spec.
servers:
  - url: http://localhost:8080
    description: local dev
paths:
  /healthz:
    get:
      operationId: getHealth
      summary: Health check
      tags:
        - system
      responses:
        '200':
          description: service is healthy
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Health'
        '500':
          description: service is unhealthy
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
components:
  schemas:
    Health:
      type: object
      required:
        - status
      properties:
        status:
          type: string
          enum: [ok]
        version:
          type: string
    Error:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: string
        message:
          type: string
```

- [ ] **Step 2: 写 `oapi-codegen.yaml`**

文件路径:`oapi-codegen.yaml`

oapi-codegen 不支持单个配置文件多段输出,所以这里给的是"基线配置",`scripts/gen.sh` 会在调用时按需覆盖 `output` 和 `package`。

```yaml
# Base output package name (overridden per-invocation in scripts/gen.sh)
package: api

# Generate: models, gin server (strict), and client.
# Each invocation in scripts/gen.sh picks the right combination.
generate:
  - models
  - gin-server
  - strict-server
  - client

# Compatibility settings
compatibility:
  apply-chi-middleware-first-to-last: false

output:
  # Default output (overridden by -o flag at runtime)
  out: internal/api/server.gen.go

output-options:
  skip-prune: false
  user-templates:
    # empty: use built-in templates
```

- [ ] **Step 3: 写 `scripts/gen.sh`**

文件路径:`scripts/gen.sh`

```bash
#!/usr/bin/env bash
# Generate server types, gin server stub, and client SDK from spec/openapi.yaml.
# Outputs:
#   internal/api/types.gen.go   (server-side types)
#   internal/api/server.gen.go  (gin server + StrictServerInterface)
#   pkg/api/types.gen.go        (client-side types)
#   pkg/api/client.gen.go       (client SDK)
set -euo pipefail

cd "$(dirname "$0")/.."

SPEC="spec/openapi.yaml"
CONFIG="oapi-codegen.yaml"
TOOL="github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"

# Ensure the binary is available
if ! command -v oapi-codegen >/dev/null 2>&1; then
  echo "Installing oapi-codegen..."
  go install "$TOOL@latest"
fi

mkdir -p internal/api pkg/api

echo "[1/4] generating internal/api/types.gen.go (models)"
oapi-codegen --config "$CONFIG" -generate models -o internal/api/types.gen.go -package api "$SPEC"

echo "[2/4] generating internal/api/server.gen.go (gin-server + strict-server)"
oapi-codegen --config "$CONFIG" -generate 'gin-server,strict-server' -o internal/api/server.gen.go -package api "$SPEC"

echo "[3/4] generating pkg/api/types.gen.go (client-side models)"
oapi-codegen --config "$CONFIG" -generate models -o pkg/api/types.gen.go -package api "$SPEC"

echo "[4/4] generating pkg/api/client.gen.go (client)"
oapi-codegen --config "$CONFIG" -generate client -o pkg/api/client.gen.go -package api "$SPEC"

echo "Done."
```

- [ ] **Step 4: 给脚本可执行权限**

Run: `chmod +x scripts/gen.sh`
Expected: 无输出

- [ ] **Step 5: 验证 spec 是合法 YAML**

Run:
```bash
python3 -c "import yaml; yaml.safe_load(open('spec/openapi.yaml'))" 2>/dev/null && echo OK || node -e "require('fs').readFileSync('spec/openapi.yaml')"
```
Expected: `OK`(若 python3 不可用,可跳过——后续 oapi-codegen 会做完整校验)

- [ ] **Step 6: Commit**

```bash
git add spec/ oapi-codegen.yaml scripts/
git commit -m "chore: add OAS spec skeleton and oapi-codegen config"
```

---

## Task 4: 运行代码生成 + 验证产物

**Files:**
- Generated: `internal/api/types.gen.go`
- Generated: `internal/api/server.gen.go`
- Generated: `pkg/api/types.gen.go`
- Generated: `pkg/api/client.gen.go`
- Modified: `go.mod` / `go.sum`(添加依赖)

**Interfaces:**
- Consumes: Task 3 的 spec 与配置
- Produces: 4 个 `*.gen.go` 文件 + `internal/api.StrictServerInterface`、`pkg/api.Client` 等关键类型(供 Task 5、6 使用)

- [ ] **Step 1: 运行生成脚本**

Run: `./scripts/gen.sh`
Expected: 4 行 `[N/4] generating ...` 输出,最后 `Done.`

如果报错 `module github.com/oapi-codegen/... not found` 之类,先跑:
```bash
go get github.com/oapi-codegen/runtime@latest
go get github.com/gin-gonic/gin@latest
```
然后重试。

- [ ] **Step 2: 验证 4 个文件存在**

Run: `ls -la internal/api/*.gen.go pkg/api/*.gen.go`
Expected: 看到 4 个 `.gen.go` 文件,体积非 0

- [ ] **Step 3: 验证生成的 server 接口存在**

Run: `grep "StrictServerInterface" internal/api/server.gen.go`
Expected: 至少一处匹配,形如 `type StrictServerInterface interface { ... }`,里面有 `GetHealth(ctx context.Context) (GetHealthRes, error)` 之类的方法(具体方法名取决于 oapi-codegen 版本的命名风格,但一定要有)

- [ ] **Step 4: 验证 client 类型存在**

Run: `grep "type Client" pkg/api/client.gen.go`
Expected: 至少一处匹配,`type Client struct` 或 `func NewClient(...)`

- [ ] **Step 5: 同步依赖**

Run:
```bash
go mod tidy
```
Expected: 把 gin、oapi-codegen/runtime 等加进 `go.sum`

- [ ] **Step 6: 验证 build 仍然通过**

Run: `go build ./...`
Expected: 无输出(成功)

如果 build 失败,可能是 `*.gen.go` 之间类型不匹配——检查 Task 3 的 spec 是否被改过。生成代码本身的问题不应该手改,只能改 spec 然后重跑 gen。

- [ ] **Step 7: Commit**

```bash
git add internal/api/ pkg/api/ go.mod go.sum
git commit -m "chore: generate server stubs and client SDK from OAS spec"
```

---

## Task 5: handler 实现 + cmd/server 启动

**Files:**
- Create: `internal/handler/handler.go`
- Create: `internal/handler/health.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: Task 4 生成的 `internal/api.StrictServerInterface`、`internal/api.NewStrictHandler`、gin 路由注册函数(通常是 `RegisterHandlers` 或类似名)
- Produces: 一个能启动的 HTTP server,响应 `GET /healthz`

- [ ] **Step 1: 创建 handler 目录**

Run: `mkdir -p internal/handler`
Expected: 无输出

- [ ] **Step 2: 写 `internal/handler/handler.go`**

文件路径:`internal/handler/handler.go`

```go
// Package handler implements the StrictServerInterface generated from OAS.
package handler

// Handler implements internal/api.StrictServerInterface.
type Handler struct{}

// New returns a new Handler.
func New() *Handler {
	return &Handler{}
}
```

> 注意:这里不写方法,方法按 OAS operation 分到独立文件(如 `health.go`)。具体方法签名见 Task 4 Step 3 grep 到的 `StrictServerInterface` 定义。

- [ ] **Step 3: 看 StrictServerInterface 的真实签名**

Run: `sed -n '/type StrictServerInterface/,/^}/p' internal/api/server.gen.go`
Expected: 打印接口定义,记下 `GetHealth`(或对应名字)的完整签名

> 模板里的方法是 `GetHealth(ctx context.Context) (GetHealthRes, error)`,但 oapi-codegen 不同版本生成的命名可能略有差异,以你看到的为准。

- [ ] **Step 4: 写 `internal/handler/health.go`**

文件路径:`internal/handler/health.go`

下面假设签名为 `GetHealth(ctx context.Context) (GetHealthRes, error)`、`GetHealthRes` 是 union wrapper(200 用 `*GetHealthOK`,500 用 `*GetHealthInternalServerError`),命名以 Task 4 Step 3 实际看到为准,如有差异在等价位置替换。

```go
package handler

import (
	"context"

	"github.com/piwriw/oas-go-template/internal/api"
)

// GetHealth implements api.StrictServerInterface.GetHealth.
func (h *Handler) GetHealth(ctx context.Context) (api.GetHealthRes, error) {
	return &api.GetHealthOK{Status: "ok", Version: "0.1.0"}, nil
}
```

> 如果生成代码里的 union wrapper 命名不同(比如 `GetHealth200JSONResponse`),用对应名字替换。原则:handler 永远 return 生成的类型,不要自己定义。

- [ ] **Step 5: 验证 handler 实现了接口**

Run:
```bash
cat > /tmp/check_impl.go <<'EOF'
//go:build ignore

package main

import (
	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/handler"
)

var _ api.StrictServerInterface = (*handler.Handler)(nil)

func main() {}
EOF
go run /tmp/check_impl.go
```

Expected: 无输出(说明 Handler 实现了接口,编译通过)

如果报错 `missing method X`,把对应的方法补到 handler 里。补的时候只调用生成的类型,不要新写业务。

- [ ] **Step 6: 改写 `cmd/server/main.go`**

文件路径:`cmd/server/main.go`

下面用 oapi-codegen 默认 gin-server 的常见 API。如果生成代码里函数名不同(比如 `RegisterHandlersWithOptions`),以生成代码为准。

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/handler"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	h := handler.New()
	strictHandler := api.NewStrictHandler(h, nil)

	r := gin.Default()
	api.RegisterHandlers(r, strictHandler)

	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
```

- [ ] **Step 7: 验证 build 通过**

Run: `go build ./...`
Expected: 无输出

如果报错 `undefined: api.NewStrictHandler` 或 `undefined: api.RegisterHandlers`,去 `internal/api/server.gen.go` 里 grep 真实导出的函数名,替换对应行:
```bash
grep -E "^(func|type)" internal/api/server.gen.go
```

- [ ] **Step 8: 启动 server,验证 healthz**

后台启动:
```bash
go run ./cmd/server &
SERVER_PID=$!
sleep 2
curl -s http://localhost:8080/healthz
kill $SERVER_PID
```

Expected: 200 响应 `{"status":"ok","version":"0.1.0"}`

如果 curl 返回的不是 200,看 server 日志排查。如果是接口/路由没匹配,通常是 handler 没正确注册——检查 `api.RegisterHandlers` 调用。

- [ ] **Step 9: Commit**

```bash
git add internal/handler/ cmd/server/
git commit -m "feat(server): implement healthz handler and wire gin"
```

---

## Task 6: cmd/client 调用示例

**Files:**
- Modify: `cmd/client/main.go`

**Interfaces:**
- Consumes: Task 4 生成的 `pkg/api.NewClient`(或 `RequestEditorFn` 等)和 `pkg/api.GetHealth` 参数类型
- Produces: 一个能跑通的 client,演示 SDK 用法

- [ ] **Step 1: 看 client 真实导出**

Run: `grep -E "^func (NewClient|.*Request)" pkg/api/client.gen.go | head -20`
Expected: 看到 `NewClient(server string, opts ...ClientOption) (*Client, error)` 之类的签名

记下 `NewClient` 的签名(第一参数一般是 server URL string)。

- [ ] **Step 2: 改写 `cmd/client/main.go`**

文件路径:`cmd/client/main.go`

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/piwriw/oas-go-template/pkg/api"
)

func main() {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	c, err := api.NewClient(serverURL)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Call GetHealth (generated method name; adjust if oapi-codegen output differs).
	resp, err := c.GetHealth(ctx)
	if err != nil {
		log.Fatalf("get health: %v", err)
	}

	switch r := resp.(type) {
	case *api.GetHealthOK:
		fmt.Printf("health: status=%s version=%s\n", r.Status, r.Version)
	case *api.GetHealthInternalServerError:
		fmt.Printf("health: unhealthy code=%s\n", r.JSONDefault.Code)
	default:
		fmt.Printf("health: unexpected response type %T\n", resp)
	}
}
```

> `c.GetHealth` 的方法名、`GetHealthOK` / `GetHealthInternalServerError` 类型名以 `pkg/api/client.gen.go` 里实际生成的为准。如果不同,grep 出来替换:
> ```bash
> grep -E "^func \(c \*Client\)" pkg/api/client.gen.go
> grep -E "^type GetHealth" pkg/api/types.gen.go pkg/api/client.gen.go
> ```

- [ ] **Step 3: 验证 build**

Run: `go build ./...`
Expected: 无输出

- [ ] **Step 4: 端到端验证(server + client)**

开两个终端(或后台进程):

```bash
go run ./cmd/server &
SERVER_PID=$!
sleep 2
go run ./cmd/client
kill $SERVER_PID
```

Expected: client 输出 `health: status=ok version=0.1.0`

- [ ] **Step 5: Commit**

```bash
git add cmd/client/
git commit -m "feat(client): add example client invocation using pkg/api SDK"
```

---

## Task 7: config 加载 + middleware

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/middleware/middleware.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: Task 5 的 server 入口
- Produces: config 模块(`NewFromEnv()`)、middleware 模块(`Recovery()`、`Logger()`),server 启动时使用

- [ ] **Step 1: 创建目录**

Run: `mkdir -p internal/config internal/middleware`
Expected: 无输出

- [ ] **Step 2: 写 `internal/config/config.go`**

文件路径:`internal/config/config.go`

```go
// Package config loads runtime configuration from environment.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the server.
type Config struct {
	HTTPAddr string
	GinMode  string
}

// NewFromEnv reads configuration from environment variables.
func NewFromEnv() (*Config, error) {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = "debug"
	}

	if err := validateMode(mode); err != nil {
		return nil, err
	}

	return &Config{HTTPAddr: addr, GinMode: mode}, nil
}

func validateMode(mode string) error {
	switch mode {
	case "debug", "release", "test":
		return nil
	default:
		return fmt.Errorf("invalid GIN_MODE %q (want debug|release|test)", mode)
	}
}

// EnvBool helper for downstream code (used by middleware, etc.).
func EnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
```

- [ ] **Step 3: 写 `internal/middleware/middleware.go`**

文件路径:`internal/middleware/middleware.go`

```go
// Package middleware bundles project-wide gin middlewares.
package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger logs each request with method, path, status, latency.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s %d %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	}
}
```

> 注意:`gin.Default()` 已经自带 `gin.Recovery()` + `gin.Logger()`。模板里我们改成 `gin.New()` + 显式注册自己的,以便后续替换/扩展。

- [ ] **Step 4: 改写 `cmd/server/main.go` 使用 config + middleware**

文件路径:`cmd/server/main.go`

```go
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/middleware"
)

func main() {
	cfg, err := config.NewFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	gin.SetMode(cfg.GinMode)

	h := handler.New()
	strictHandler := api.NewStrictHandler(h, nil)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.Logger())
	api.RegisterHandlers(r, strictHandler)

	log.Printf("server listening on %s (mode=%s)", cfg.HTTPAddr, cfg.GinMode)
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
```

- [ ] **Step 5: 验证 build**

Run: `go build ./...`
Expected: 无输出

- [ ] **Step 6: 跑 server,看自定义日志**

```bash
go run ./cmd/server &
SERVER_PID=$!
sleep 2
curl -s http://localhost:8080/healthz
sleep 1
kill $SERVER_PID
```

Expected:
- 200 响应 `{"status":"ok","version":"0.1.0"}`
- server 日志有一行类似 `GET /healthz 200 123.4µs`

- [ ] **Step 7: 验证 config 校验生效**

Run:
```bash
GIN_MODE=invalid go run ./cmd/server 2>&1 | head -3
```

Expected: 程序退出,输出包含 `config: invalid GIN_MODE "invalid"`

- [ ] **Step 8: Commit**

```bash
git add internal/config/ internal/middleware/ cmd/server/
git commit -m "feat(server): add config and middleware modules"
```

---

## Task 8: 前端 `web/` 初始化(Vite + React + TypeScript)

**Files:**
- Create: `web/`(整个 Vite 默认项目)
- Modify: 根 `.gitignore`(已在 Task 1 处理过 node_modules,无需改)

**Interfaces:**
- Consumes: 无
- Produces: 一个能 `npm run dev` 跑起来的 React 应用

- [ ] **Step 1: 用 Vite 创建项目**

Run:
```bash
npm create vite@latest web -- --template react-ts
```

Expected: 终端无报错,`web/` 下出现 `package.json`、`src/`、`public/`、`vite.config.ts`、`tsconfig.json` 等

- [ ] **Step 2: 安装依赖**

Run:
```bash
cd web && npm install
```

Expected: 依赖安装成功,生成 `node_modules/`(被 .gitignore 排除)、`package-lock.json`

- [ ] **Step 3: 验证 build**

Run:
```bash
cd web && npm run build
```

Expected: `dist/` 目录生成,有 `index.html` 和 `assets/` 子目录

- [ ] **Step 4: 创建前端 README 占位**

文件路径:`web/README.md`(若 Vite 已生成,在其后追加一节)

```markdown
# oas-go-template web

Frontend SPA. Independent from backend; deploys separately to CDN/Nginx.

## Dev

```bash
npm install
npm run dev
```

## Build

```bash
npm run build
# outputs to dist/
```

## API Client

前端 OAS client 不在本模板初始范围内,预留 `src/api/` 目录占位。
后续如需,可基于 `../spec/openapi.yaml` 用 `openapi-typescript` 或 `openapi-fetch` 生成前端 client。
```

- [ ] **Step 5: 创建 `src/api/.gitkeep` 占位**

Run: `mkdir -p web/src/api && touch web/src/api/.gitkeep`
Expected: 无输出

- [ ] **Step 6: 验证 dist 已被 .gitignore**

Run: `git check-ignore web/dist web/node_modules`
Expected: 两行都被打印(说明被忽略)

- [ ] **Step 7: Commit**

```bash
git add web/
git commit -m "chore(web): init Vite + React + TypeScript scaffold"
```

---

## Task 9: Dockerfile + .dockerignore

**Files:**
- Create: `build/Dockerfile`
- Create: `.dockerignore`

**Interfaces:**
- Consumes: Task 5 的 `cmd/server` 二进制构建产物
- Produces: 一个能用 `docker build` 构建的最小后端镜像

- [ ] **Step 1: 写 `build/Dockerfile`**

文件路径:`build/Dockerfile`

多阶段构建。Stage 1 编译静态二进制,Stage 2 用 alpine 跑。

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a static binary
# CGO_DISABLED=1 + -ldflags '-extldflags "-static"' for static link
# trimpath removes absolute paths for reproducibility
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags='-s -w -extldflags "-static"' \
    -o /out/server \
    ./cmd/server

# Runtime stage
FROM alpine:3.20

# ca-certificates for HTTPS calls, tzdata for time zone support
RUN apk --no-cache add ca-certificates tzdata

# Non-root user
RUN adduser -D -u 10001 app
USER app

WORKDIR /app
COPY --from=builder /out/server /app/server

ENV HTTP_ADDR=:8080
EXPOSE 8080

ENTRYPOINT ["/app/server"]
```

- [ ] **Step 2: 写 `.dockerignore`**

文件路径:`.dockerignore`

```
# VCS
.git
.gitignore

# IDE
.idea
.vscode

# Build artifacts
/bin
/dist
/web/dist
/web/node_modules

# Docs / specs not needed at runtime
/docs
*.md
!README.md

# Misc
.env
.env.*
.DS_Store
tmp/
```

- [ ] **Step 3: 验证 docker build 成功**

Run:
```bash
docker build -f build/Dockerfile -t oas-go-template:dev .
```

Expected: 构建成功,最后 `Successfully tagged oas-go-template:dev`

如果失败,看错误信息。常见问题:
- `go mod download` 失败 → 网络问题
- `CGO_ENABLED=0` 报错 → 工具链问题,确认 Go 1.23+

- [ ] **Step 4: 验证容器可运行**

Run:
```bash
docker run --rm -d -p 18080:8080 --name oas-template-smoke oas-go-template:dev
sleep 2
curl -s http://localhost:18080/healthz
docker stop oas-template-smoke
```

Expected: 200 响应 `{"status":"ok","version":"0.1.0"}`

- [ ] **Step 5: Commit**

```bash
git add build/Dockerfile .dockerignore
git commit -m "chore(build): add multi-stage Dockerfile for server"
```

---

## Task 10: golangci-lint + Makefile

**Files:**
- Create: `.golangci.yml`
- Create: `Makefile`

**Interfaces:**
- Consumes: 之前所有任务
- Produces: 统一的 lint 规则与开发命令入口

- [ ] **Step 1: 写 `.golangci.yml`**

文件路径:`.golangci.yml`

```yaml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  default: standard
  enable:
    - errcheck
    - gocritic
    - govet
    - ineffassign
    - revive
    - staticcheck
    - unused
    - misspell

issues:
  exclude-rules:
    # Don't lint generated code
    - path: '.*\.gen\.go$'
      linters:
        - all
  max-issues-per-linter: 0
  max-same-issues: 0
```

> 注意:golangci-lint v2 用 `linters.default: standard` + `enable: [...]` 增量启用。如果你装的是 v1.x,语法是 `linters.enable: [...]`,没有 `default` 字段。本计划假设 v2(本地装的是 2.12.2)。

- [ ] **Step 2: 跑 lint,确认生成代码被排除**

Run: `golangci-lint run`
Expected: 无输出(或只有警告但不是 `*.gen.go` 相关)

如果报错 `unknown linter: XXX`,移除该 linter 名(不同版本支持集合不同)。
如果 lint 报错 `*.gen.go` 里有问题,检查 `exclude-rules` 的 `path` 正则。

- [ ] **Step 3: 写 `Makefile`**

文件路径:`Makefile`

```makefile
# oas-go-template Makefile
.PHONY: help gen build run run-client test lint docker dev clean web-dev web-build

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

gen:  ## Generate code from spec/openapi.yaml
	./scripts/gen.sh

build:  ## Build server and client binaries into ./bin
	mkdir -p bin
	go build -o bin/server ./cmd/server
	go build -o bin/client ./cmd/client

run:  ## Run server locally
	go run ./cmd/server

run-client:  ## Run client locally (assumes server is up)
	go run ./cmd/client

test:  ## Run all tests
	go test -race -cover ./...

lint:  ## Run golangci-lint
	golangci-lint run

docker:  ## Build server docker image
	docker build -f build/Dockerfile -t oas-go-template:latest .

dev:  ## Run server with live reload (requires air: go install github.com/air-verse/air@latest)
	air

web-dev:  ## Run frontend dev server
	cd web && npm run dev

web-build:  ## Build frontend
	cd web && npm run build

clean:  ## Remove build artifacts
	rm -rf bin web/dist
```

> 注意:Makefile 必须用 TAB 缩进(`.editorconfig` 已声明)。如果用 spaces 会报 `*** missing separator`。

- [ ] **Step 4: 验证 make help**

Run: `make help`
Expected: 列出所有命令

- [ ] **Step 5: 验证 make build**

Run: `make build && ls bin`
Expected: `bin/` 下有 `server` 和 `client` 两个二进制

- [ ] **Step 6: 验证 make test**

Run: `make test`
Expected: `ok`(目前还没有测试用例,但应该无错误)

- [ ] **Step 7: 验证 make lint**

Run: `make lint`
Expected: 无输出(成功)

- [ ] **Step 8: Commit**

```bash
git add .golangci.yml Makefile
git commit -m "chore: add golangci-lint config and Makefile"
```

---

## Task 11: 端到端验证

**Files:**
- 无新建,只跑命令验证整套流程

**Interfaces:**
- Consumes: 所有前置任务
- Produces: 一个干净的、可工作、可发布的模板仓库

- [ ] **Step 1: 从干净状态重新跑完整流程**

```bash
git status  # 应该 clean
make gen    # 重新生成代码
git status  # 应该仍然 clean(生成的代码已经 commit 了)
```

Expected: `make gen` 跑完后 `git status` 无变化(说明生成产物稳定)

如果 `git status` 有 diff,说明生成代码不稳定(oapi-codegen 的输出在两次运行间漂移),需要:
- 检查 spec 是否被改过
- 检查 `scripts/gen.sh` 的命令是否每次都以相同顺序/参数调用
- 排查后再次 `make gen`,把稳定后的产物 commit

- [ ] **Step 2: 完整构建 + 测试 + lint**

```bash
make clean
make gen
make build
make test
make lint
```

Expected: 全部无错

- [ ] **Step 3: Docker 端到端**

```bash
make docker
docker run --rm -d -p 18080:8080 --name smoke oas-go-template:latest
sleep 2
curl -sf http://localhost:18080/healthz
docker stop smoke
```

Expected: `make docker` 成功;curl 返回 0;容器响应 healthz 200

- [ ] **Step 4: Client 调 Docker 化的 server**

```bash
docker run --rm -d -p 18080:8080 --name smoke oas-go-template:latest
sleep 2
SERVER_URL=http://localhost:18080 ./bin/client
docker stop smoke
```

Expected: client 输出 `health: status=ok version=0.1.0`

- [ ] **Step 5: 前端构建**

Run: `make web-build`
Expected: `web/dist/` 重新生成,无错误

- [ ] **Step 6: 仓库整体核对**

Run: `git log --oneline && tree -L 2 -I 'node_modules|.git|bin'`
Expected:
- 看到 9-10 个 commit(每个 Task 一个)
- 目录结构跟设计文档 §3 一致

如果系统没有 `tree` 命令,用 `find . -maxdepth 2 -not -path '*/node_modules/*' -not -path '*/.git/*'` 代替。

- [ ] **Step 7: 最终 README 校对**

打开 `README.md`,确认 Quickstart 命令都跑得通(已在前面 Step 验证)。如果文档与实际不符,修正后单独 commit。

- [ ] **Step 8: 收尾 commit(如果有 README 调整)**

```bash
git add README.md
git commit -m "docs: align README quickstart with actual make targets"
```

如无调整,跳过。

---

## 完成标志

模板项目初始化完成的判据:

- ✅ `make gen && make build && make test && make lint` 全部成功
- ✅ `make docker` 构建镜像,容器能响应 `/healthz`
- ✅ `./bin/client` 能成功调用 `./bin/server`
- ✅ `git status` 在 `make gen` 之后仍然 clean(生成代码稳定)
- ✅ `web/` 能 `npm run dev` 和 `npm run build`
- ✅ 目录结构与 `docs/superpowers/specs/2026-07-04-project-init-design.md` §3 一致
