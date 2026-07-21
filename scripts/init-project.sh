#!/usr/bin/env bash
# init-project.sh — rename this template's module path and project name in one shot.
#
# Usage:
#   ./scripts/init-project.sh <new-module-path> [new-short-name]
#
# Examples:
#   ./scripts/init-project.sh github.com/acme/widget
#   ./scripts/init-project.sh github.com/acme/widget acme-widget
#
# What it does:
#   1. Derives OLD module path and short name from go.mod (no hard-coded strings,
#      so this script is safe to re-run and won't match itself).
#   2. Replaces the module path everywhere (Go imports, Makefile ldflags,
#      Dockerfile, .golangci.yml goimports local-prefix, scripts, docs).
#   3. Replaces the short name everywhere (cmd/server serviceName, docker tags,
#      Helm chart name + helpers + README, README/CLAUDE/CONTRIBUTING titles).
#   4. Skips *.gen.go (regenerated in step 5).
#   5. Runs `make gen` so the embedded OAS + types pick up the new package.
#   6. Prints any leftovers you need to handle by hand (registry prefix to
#      prepend to chart image repos when you push to a remote registry).
#
# After it finishes: review the leftovers, set chart image repositories to your
# registry paths, edit spec/openapi.yaml to define your API, then commit.

set -euo pipefail

# ─── Args ────────────────────────────────────────────────────────────────────

if [ $# -lt 1 ] || [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
fi

NEW_MOD=$1
NEW_NAME=${2:-$(basename "$NEW_MOD")}

# ─── Derive OLD from current state ───────────────────────────────────────────

if [ ! -f go.mod ]; then
    echo "error: no go.mod in cwd — run from the project root" >&2
    exit 1
fi

OLD_MOD=$(awk '/^module / { print $2; exit }' go.mod)
OLD_NAME=$(basename "$OLD_MOD")

if [ -z "$OLD_MOD" ]; then
    echo "error: couldn't read module path from go.mod" >&2
    exit 1
fi
if [ "$OLD_MOD" = "$NEW_MOD" ]; then
    echo "already renamed: module is $NEW_MOD" >&2
    exit 0
fi

# ─── Sanity: short name shouldn't be a substring of the new module path's
# last segment in a way that would cause double-rewrite. Bail with a hint.

# Reject names that contain regex metachars — we use them in sed below.
case "$NEW_NAME" in
    *[/\&*.\[]*) echo "error: new-short-name '$NEW_NAME' has sed metachars; pick a plain identifier" >&2; exit 1 ;;
esac

echo "Rewriting:"
echo "  module:  $OLD_MOD  ->  $NEW_MOD"
echo "  name:    $OLD_NAME  ->  $NEW_NAME"
echo

# ─── File globs ──────────────────────────────────────────────────────────────

# Files that should be rewritten. Excludes *.gen.go (regenerated below),
# .git/, node_modules/, and binary artifacts.
INCLUDES=(
    --include='*.go'
    --include='*.yaml'
    --include='*.yml'
    --include='Makefile'
    --include='Dockerfile'
    --include='*.sh'
    --include='*.md'
    --include='*.tpl'
    --include='*.txt'
    --include='go.mod'
    --include='go.sum'
)
EXCLUDES=(
    --exclude-dir=.git
    --exclude-dir=node_modules
    --exclude-dir=dist
    --exclude-dir=bin
)

# Helper: rewrite OLD -> NEW across all matching files.
rewrite() {
    local old=$1 new=$2
    # shellcheck disable=SC2086
    grep -rl "$old" "${INCLUDES[@]}" "${EXCLUDES[@]}" . \
        | grep -v '\.gen\.go$' \
        | xargs -r sed -i.bak "s|${old}|${new}|g" || true
}

# 1) Module path. Goes first so the short-name pass below doesn't accidentally
#    eat a partial module path.
rewrite "$OLD_MOD" "$NEW_MOD"

# 2) Short name. Matches the bare identifier — service.name, docker tags,
#    Helm chart helpers, README title.
rewrite "$OLD_NAME" "$NEW_NAME"

# 3) Clean up sed backup files.
find . -name '*.bak' -delete

# ─── Regenerate ──────────────────────────────────────────────────────────────

# Generated code wasn't touched above. Run make gen so *.gen.go reflects the
# current spec (the spec itself wasn't changed, but this also confirms the
# generator is idempotent in this checkout).
if command -v make >/dev/null 2>&1; then
    echo
    echo "Running 'make gen' to refresh generated code..."
    make gen
fi

# ─── Verify ──────────────────────────────────────────────────────────────────

echo
echo "Checking for leftovers..."
leftovers=$(grep -rn "$OLD_MOD\|$OLD_NAME" \
    "${INCLUDES[@]}" "${EXCLUDES[@]}" . \
    | grep -v '\.gen\.go$' \
    | grep -v 'init-project.sh' \
    || true)

if [ -n "$leftovers" ]; then
    echo "WARNING: these references weren't rewritten — handle manually:" >&2
    echo "$leftovers" >&2
else
    echo "Clean — no stale references to $OLD_MOD or $OLD_NAME."
fi

# ─── Next steps ──────────────────────────────────────────────────────────────

cat <<EOF

Done. Renamed:
  module:  $OLD_MOD  ->  $NEW_MOD
  name:    $OLD_NAME  ->  $NEW_NAME

Manual follow-ups (the script can't infer these):
  1. chart/values.yaml: image repos default to "$NEW_NAME" / "${NEW_NAME}-web"
     (script rewrote them — matches the Docker tags from 'make docker' /
     'make web-docker'). Prepend your registry prefix only if you push to a
     remote, e.g.  ghcr.io/yourorg/$NEW_NAME
  2. spec/openapi.yaml: replace the example paths (/healthz, /readyz, /version)
     with your real API, then run 'make gen' again.
  3. Author / copyright: README.md (© line) and chart/Chart.yaml (maintainers)
     still name the original author — edit by hand.
  4. Git: if you haven't already,
       rm -rf .git && git init && git branch -m main
  5. Verify the full pipeline:
       make build test lint
       make docker && docker run --rm -d -p 18000:8000 --name smoke $NEW_NAME:latest

EOF
