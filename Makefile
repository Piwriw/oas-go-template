# oas-go-template Makefile
.PHONY: help gen tools contract-check supply-chain-check build run test lint lint-config lint-version-check fmt audit docker web-docker helm-lint helm-template dev clean web-dev web-build dev-stack dev-stack-down

# Build metadata injected via ldflags. Override like: make build VERSION=v1.0.0
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

VERSION_PKG := github.com/piwriw/oas-go-template/internal/version
LDFLAGS     := -X $(VERSION_PKG).Version=$(VERSION) \
               -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
               -X $(VERSION_PKG).BuildTime=$(BUILD_TIME)

# Go-managed tool versions live in go.mod. Keep tools that intentionally run
# outside the application module graph pinned here.
GOLANGCI_LINT_VERSION ?= 2.12.2
GOSEC_VERSION ?= v2.27.1
AIR_VERSION ?= v1.66.0
OASDIFF_VERSION ?= v1.10.28

# BASE_SPEC may point to the API contract from the merge base in CI. Locally,
# the default makes the target a useful no-op smoke check.
BASE_SPEC ?= spec/openapi.yaml

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

gen:  ## Generate code from spec/openapi.yaml
	./scripts/gen.sh

contract-check:  ## Reject breaking OpenAPI changes against BASE_SPEC
	go run github.com/tufin/oasdiff@$(OASDIFF_VERSION) breaking "$(BASE_SPEC)" spec/openapi.yaml --fail-on ERR

supply-chain-check:  ## Verify Go, tool, Docker image, and GitHub Action pins
	./scripts/verify-pins.sh

tools:  ## Install pinned local-only tools (Go tools run directly from go.mod)
	go install github.com/air-verse/air@$(AIR_VERSION)

build:  ## Build server binary into ./bin (with version ldflags)
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server

run:  ## Run server locally (with version ldflags)
	go run -ldflags "$(LDFLAGS)" ./cmd/server

test:  ## Run all tests
	go test -race -cover ./...

lint: lint-version-check  ## Run golangci-lint
	golangci-lint run ./...

lint-config: lint-version-check  ## Verify the golangci-lint v2 configuration
	golangci-lint config verify

lint-version-check:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found; install an official binary: https://golangci-lint.run/docs/welcome/install/local/" >&2; exit 1; }
	@actual=$$(golangci-lint version --short); test "$$actual" = "$(GOLANGCI_LINT_VERSION)" || { echo "golangci-lint $$actual found; want $(GOLANGCI_LINT_VERSION)" >&2; exit 1; }

fmt:  ## Format Go code with goimports (gofmt + import grouping)
	go tool goimports -local github.com/piwriw/oas-go-template -w .

audit:  ## Scan dependencies (govulncheck) and source (gosec) for security issues
	go tool govulncheck ./...
	go run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) -quiet ./...

docker:  ## Build server docker image (override GOPROXY via env if behind restricted network)
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg GIT_COMMIT=$(GIT_COMMIT) \
	  --build-arg BUILD_TIME=$(BUILD_TIME) \
	  -f build/Dockerfile \
	  $(if $(GOPROXY),--build-arg GOPROXY=$(GOPROXY)) \
	  -t oas-go-template:latest .

web-docker:  ## Build frontend docker image (multi-stage: node build → nginx serve)
	docker build -f web/Dockerfile -t oas-go-template-web:latest web/

helm-lint:  ## Lint the Helm chart (requires helm 3)
	helm lint chart/

helm-template:  ## Render chart templates locally for inspection (no cluster needed)
	helm template oas-go-template chart/ | less

dev:  ## Run server with live reload (run make tools first)
	air

web-dev:  ## Run frontend dev server
	cd web && npm run dev

web-build:  ## Build frontend
	cd web && npm run build

clean:  ## Remove build artifacts
	rm -rf bin web/dist

dev-stack:  ## Start local OTel collector + Jaeger (docker compose up -d)
	docker compose up -d

dev-stack-down:  ## Stop local OTel collector + Jaeger
	docker compose down
