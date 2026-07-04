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

docker:  ## Build server docker image (override GOPROXY via env if behind restricted network)
	docker build -f build/Dockerfile $(if $(GOPROXY),--build-arg GOPROXY=$(GOPROXY)) -t oas-go-template:latest .

dev:  ## Run server with live reload (requires air: go install github.com/air-verse/air@latest)
	air

web-dev:  ## Run frontend dev server
	cd web && npm run dev

web-build:  ## Build frontend
	cd web && npm run build

clean:  ## Remove build artifacts
	rm -rf bin web/dist
