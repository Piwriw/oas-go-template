#!/usr/bin/env bash
# Generate server types, gin server stub, client SDK, and embedded OAS document from spec/openapi.yaml.
# Outputs:
#   internal/api/types.gen.go   (server-side types — data models)
#   internal/api/spec.gen.go    (gin bindings + StrictServerInterface — the contract)
#   pkg/api/types.gen.go        (client-side types)
#   pkg/api/client.gen.go       (client SDK)
#   pkg/api/spec.gen.go         (embedded OAS document — GetSpec / GetSpecJSON for runtime introspection)
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

echo "[1/5] generating internal/api/types.gen.go (models)"
oapi-codegen --config "$CONFIG" -generate models -o internal/api/types.gen.go -package api "$SPEC"

echo "[2/5] generating internal/api/spec.gen.go (gin-server + strict-server)"
oapi-codegen --config "$CONFIG" -generate 'gin-server,strict-server' -o internal/api/spec.gen.go -package api "$SPEC"

echo "[3/5] generating pkg/api/types.gen.go (client-side models)"
oapi-codegen --config "$CONFIG" -generate models -o pkg/api/types.gen.go -package api "$SPEC"

echo "[4/5] generating pkg/api/client.gen.go (client)"
oapi-codegen --config "$CONFIG" -generate client -o pkg/api/client.gen.go -package api "$SPEC"

echo "[5/5] generating pkg/api/spec.gen.go (embedded OAS document)"
oapi-codegen --config "$CONFIG" -generate spec -o pkg/api/spec.gen.go -package api "$SPEC"

echo "Done."
