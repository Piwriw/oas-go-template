---
name: Feature request
about: Suggest something the template should do (or do better)
title: "[feat] "
labels: enhancement
body:
  - type: textarea
    id: problem
    attributes:
      label: What problem are you trying to solve?
      description: The use case, not the proposed solution. "I want to X so that Y."
    validations:
      required: true
  - type: textarea
    id: proposal
    attributes:
      label: What would the ideal solution look like?
      description: A sketch is fine — code, config snippets, or just bullet points.
  - type: textarea
    id: alternatives
    attributes:
      label: Alternatives considered
      description: What else did you try or think about? Why is it worse?
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
        - Build / Makefile / CI
        - Other
    validations:
      required: true
---
