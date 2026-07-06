# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in `oas-go-template`, **do not open
a public issue**. Use GitHub's private vulnerability reporting instead:

**[Report a vulnerability](https://github.com/piwriw/oas-go-template/security/advisories/new)**

Please include:

- A description of the vulnerability and its impact
- Steps to reproduce (proof-of-concept is ideal)
- Affected versions / commits
- Suggested fix, if any

You should receive an initial response within 72 hours. Confirmed
vulnerabilities will be coordinated through a GitHub Security Advisory and
credited in the release notes (anonymous by request).

## Scope

This policy covers the `piwriw/oas-go-template` repository only. Projects
derived from this template should establish their own security policy and
contact paths.

## Pre-disclosure checks

Before reporting, run the local audit to confirm the finding is reachable
in the current codebase:

```bash
make audit   # govulncheck + gosec
```

If `make audit` already flags the dependency or pattern you're reporting,
mention that in your report — it speeds up triage.

## Supported versions

Only the latest minor release receives security fixes.

| Version | Supported |
|---------|-----------|
| latest  | ✅         |
| older   | ❌         |
