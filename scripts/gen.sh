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

repo_root="$PWD"
spec="$repo_root/spec/openapi.yaml"

mkdir -p internal/api pkg/api

# Run in the isolated generator module; all input and output paths are absolute.
cd tools

echo "[1/5] generating internal/api/types.gen.go (models)"
go tool oapi-codegen --config "$repo_root/spec/models.cfg.yaml" -o "$repo_root/internal/api/types.gen.go" "$spec"

echo "[2/5] generating internal/api/spec.gen.go (gin-server + strict-server)"
go tool oapi-codegen --config "$repo_root/spec/server.cfg.yaml" -o "$repo_root/internal/api/spec.gen.go" "$spec"

echo "[3/5] generating pkg/api/types.gen.go (client-side models)"
go tool oapi-codegen --config "$repo_root/spec/models.cfg.yaml" -o "$repo_root/pkg/api/types.gen.go" "$spec"

echo "[4/5] generating pkg/api/client.gen.go (client)"
go tool oapi-codegen --config "$repo_root/spec/client.cfg.yaml" -o "$repo_root/pkg/api/client.gen.go" "$spec"

echo "[5/5] generating pkg/api/spec.gen.go (embedded OAS document)"
go tool oapi-codegen --config "$repo_root/spec/embedded.cfg.yaml" -o "$repo_root/pkg/api/spec.gen.go" "$spec"

echo "Done."
