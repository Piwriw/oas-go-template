# Contributing

Thanks for considering a contribution! This is a template repo, so most
"contributions" are tweaks to the boilerplate itself — keep PRs focused and
small.

## Setup

Requirements:

- Go 1.25+
- [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) v2.7.1 —
  installed by `scripts/gen.sh`, no separate install needed
- `make`, `docker`, `helm` (only for chart changes), Node 22+ (only for `web/`)
- Pinned developer tools — install with `make tools`

```bash
git clone <this repo>
cd oas-go-template
cp config.example.yaml config.yaml
make gen build test
```

## Daily workflow

```bash
make gen        # regenerate *.gen.go after editing spec/openapi.yaml
make tools      # install pinned oapi-codegen, golangci-lint, and air
# OAPI_CODEGEN_VERSION=vX.Y.Z make gen  # coordinated generator upgrade only
make lint test  # always green before pushing
make audit      # govulncheck v1.6.0 + gosec v2.27.1
```

## Code generation rules

- `spec/openapi.yaml` is the **single source of truth**. Never hand-edit
  `*.gen.go` — they are committed only so reviewers and IDEs see what compiles.
- After editing the spec, run `make gen` and commit the regenerated files in
  the same PR.
- If `make gen` produces a diff on a clean tree, generation isn't idempotent —
  fix the spec or the generator config before opening a PR.

## API versioning and deprecation

- Keep `/healthz`, `/readyz`, and `/version` unversioned; they are operational
  probe contracts. Put every new business path under `/vN/`.
- For a deprecated operation, set `deprecated: true`,
  `x-deprecation-date`, and `x-sunset-date` to RFC3339 timestamps. The sunset
  must be later than the deprecation date. Runtime middleware emits matching
  `Deprecation` and `Sunset` response headers.
- Do not remove or make an existing operation stricter without either adding a
  new `/vN` contract or documenting an approved migration. PR CI runs the
  pinned `oasdiff` check against the target branch's OpenAPI document.

For a local check, provide the prior contract explicitly:

```bash
make contract-check BASE_SPEC=/path/to/openapi-base.yaml
```

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(api): add /orders endpoint
fix(handler): return 503 when db ping fails
docs: clarify config loading order
chore(deps): bump otel to v1.41.0
```

These prefixes feed release notes and any future automated tooling, so
consistency matters.

## Sign-off (DCO)

Every commit must carry a `Signed-off-by:` trailer — this is your
attestation to the [Developer Certificate of Origin](https://developercertificate.org/)
that you wrote the change and have the right to contribute it.

```bash
git commit -s   # or --signoff — adds the trailer using git config user.name / user.email
```

The CI `Sign-off Check (DCO)` job rejects any PR commit missing the
trailer. To fix existing commits:

```bash
git rebase --signoff origin/main
git push --force-with-lease
```

The trailer takes the form `Signed-off-by: Your Name <you@example.com>`
and must match your git identity.

## PR checklist

- [ ] `make lint test audit` passes locally
- [ ] `make gen` produces no diff (codegen is idempotent)
- [ ] `make contract-check BASE_SPEC=/path/to/openapi-base.yaml` passes, or
      the PR explains the version/migration strategy for an intentional break
- [ ] New endpoints have handler implementations, not just generated stubs
- [ ] No secrets, real DSNs, or customer data in commits
- [ ] If you changed `spec/openapi.yaml`, the regenerated `*.gen.go` are
      committed in the same PR
- [ ] Every commit has a `Signed-off-by:` trailer (see [Sign-off (DCO)](#sign-off-dco))

## Reporting bugs

Open an issue using the **Bug report** template. Include the `make build`
output, your Go version, and the smallest `spec/openapi.yaml` snippet that
reproduces the issue.

## Security disclosures

**Do not file a public issue for security vulnerabilities.** See
[SECURITY.md](SECURITY.md) for the private reporting channel
(GitHub Security Advisory). Run `make audit` locally to confirm a finding
before reporting.

## License

By contributing you agree your changes will be licensed under the [MIT
License](LICENSE), the same terms as the rest of this repository.
