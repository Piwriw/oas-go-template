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
