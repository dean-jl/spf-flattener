name: 🐞 Bug Report
description: Report a reproducible issue or unexpected behavior
title: "[Bug] "
labels: [bug]
assignees: []

body:
- type: markdown
  attributes:
  value: |
  Thanks for reporting a bug! Please fill out the details below to help us reproduce and understand the issue.

- type: input
  id: summary
  attributes:
  label: Brief Summary
  description: What’s going wrong?
  placeholder: e.g., SPF record not flattening correctly for domain X

- type: textarea
  id: steps
  attributes:
  label: Steps to Reproduce
  description: Include CLI commands, config snippets, and any relevant flags
  placeholder: |
  1. Run `spf-flattener flatten --config config.yaml`
  2. Observe output or error message

- type: textarea
  id: expected
  attributes:
  label: Expected Behavior
  description: What did you expect to happen?

- type: textarea
  id: actual
  attributes:
  label: Actual Behavior
  description: What actually happened?

- type: input
  id: version
  attributes:
  label: Tool Version
  description: Run `spf-flattener --version` and paste the result
  placeholder: e.g., v2.0.0

- type: textarea
  id: environment
  attributes:
  label: Environment Details
  description: OS, shell, DNS provider, and any other relevant info
  placeholder: e.g., Ubuntu 22.04, Bash, Porkbun
