# Contributing

Thanks for considering a contribution! This is a template repo, so most
"contributions" are tweaks to the boilerplate itself — keep PRs focused and
small.

## Setup

Requirements:

- Go 1.25+
- [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) — pulled via
  `scripts/gen.sh` using `go run`, no separate install needed
- `make`, `docker`, `helm` (only for chart changes), Node 22+ (only for `web/`)
- `golangci-lint` v2 — install via
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`

```bash
git clone <this repo>
cd oas-go-template
cp config.example.yaml config.yaml
make gen build test
```

## Daily workflow

```bash
make gen        # regenerate *.gen.go after editing spec/openapi.yaml
make lint test  # always green before pushing
make audit      # govulncheck + gosec, exits non-zero on findings
```

## Code generation rules

- `spec/openapi.yaml` is the **single source of truth**. Never hand-edit
  `*.gen.go` — they are committed only so reviewers and IDEs see what compiles.
- After editing the spec, run `make gen` and commit the regenerated files in
  the same PR.
- If `make gen` produces a diff on a clean tree, generation isn't idempotent —
  fix the spec or the generator config before opening a PR.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(api): add /orders endpoint
fix(handler): return 503 when db ping fails
docs: clarify config loading order
chore(deps): bump otel to v1.41.0
```

Goreleaser filters the changelog by these prefixes, so consistency matters.

## PR checklist

- [ ] `make lint test audit` passes locally
- [ ] `make gen` produces no diff (codegen is idempotent)
- [ ] New endpoints have handler implementations, not just generated stubs
- [ ] No secrets, real DSNs, or customer data in commits
- [ ] If you changed `spec/openapi.yaml`, the regenerated `*.gen.go` are
      committed in the same PR

## Reporting bugs

Open an issue using the **Bug report** template. Include the `make build`
output, your Go version, and the smallest `spec/openapi.yaml` snippet that
reproduces the issue.

## Security disclosures

**Do not file a public issue for security vulnerabilities.** Email the
maintainer directly instead. Run `make audit` locally to confirm a finding
before reporting.

## License

By contributing you agree your changes will be licensed under the [MIT
License](LICENSE), the same terms as the rest of this repository.
