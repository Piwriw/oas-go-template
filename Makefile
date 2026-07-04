# oas-go-template Makefile
.PHONY: help gen build run run-client test lint fmt audit docker web-docker helm-lint helm-template dev clean web-dev web-build dev-stack dev-stack-down

# Build metadata injected via ldflags. Override like: make build VERSION=v1.0.0
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

VERSION_PKG := github.com/piwriw/oas-go-template/internal/version
LDFLAGS     := -X $(VERSION_PKG).Version=$(VERSION) \
               -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
               -X $(VERSION_PKG).BuildTime=$(BUILD_TIME)

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

gen:  ## Generate code from spec/openapi.yaml
	./scripts/gen.sh

build:  ## Build server and client binaries into ./bin (with version ldflags)
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server
	go build -ldflags "$(LDFLAGS)" -o bin/client ./cmd/client

run:  ## Run server locally (with version ldflags)
	go run -ldflags "$(LDFLAGS)" ./cmd/server

run-client:  ## Run client locally (assumes server is up)
	go run ./cmd/client

test:  ## Run all tests
	go test -race -cover ./...

lint:  ## Run golangci-lint
	golangci-lint run

fmt:  ## Format Go code with goimports (gofmt + import grouping)
	goimports -local github.com/piwriw/oas-go-template -w .

audit:  ## Scan dependencies (govulncheck) and source (gosec) for security issues
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go run github.com/securego/gosec/v2/cmd/gosec@latest -quiet ./...

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

dev:  ## Run server with live reload (requires air: go install github.com/air-verse/air@latest)
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
