---
name: Bug report
about: Something isn't working as documented
title: "[bug] "
labels: bug
body:
  - type: textarea
    id: what-happened
    attributes:
      label: What happened?
      description: A clear description of the unexpected behavior. Include the smallest spec / config snippet that reproduces it.
      placeholder: |
        Steps to reproduce:
        1. ...
        2. ...

        Expected: ...
        Actual: ...
    validations:
      required: true
  - type: input
    id: version
    attributes:
      label: Version
      description: Output of `./bin/server -version` or the git commit you built from.
      placeholder: "v0.1.0 / commit abc1234 / dev"
    validations:
      required: true
  - type: input
    id: go-version
    attributes:
      label: Go version
      description: Output of `go version`.
      placeholder: "go1.25.0 darwin/arm64"
    validations:
      required: true
  - type: textarea
    id: logs
    attributes:
      label: Logs
      description: Relevant log lines with `trace_id` / `span_id` if you have them. Paste as-is.
      render: shell
  - type: dropdown
    id: area
    attributes:
      label: Affected area
      options:
        - Code generation (oapi-codegen)
        - HTTP handler / StrictServerInterface
        - Config loading
        - Database (Gorm)
        - OpenTelemetry / logging
        - Helm chart
        - Frontend (web/)
        - Other
    validations:
      required: true
---

<!-- For security vulnerabilities, do NOT file a public issue. Email the maintainer directly. -->
