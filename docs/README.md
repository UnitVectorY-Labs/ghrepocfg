---
layout: default
title: ghrepocfg
nav_order: 1
permalink: /
---

# ghrepocfg

Declaratively export, review, layer, compare, and reconcile GitHub repository settings from portable YAML files.

**ghrepocfg** turns repository governance into a code-reviewable contract. It manages repository behavior, custom properties, security features, GitHub Actions policy, direct collaborators, team access, repository rulesets, deployment environments, Pages, labels, autolinks, and deploy keys while keeping every proposed change visible before it is applied.

Manage deployment approvals, cloud identity claims, build retention, and release controls alongside the settings your team already reviews. Export readable variables and publishing settings with the same explicit management boundary.

## Key Features

- **Desired-state configuration** — only fields present in YAML are managed
- **Complete drift plans** — inspect every addition, modification, and removal before applying
- **Safe access management** — reconcile direct collaborators, pending invitations, and team permissions authoritatively
- **Repository rulesets** — manage branch, tag, and push rulesets without converting legacy protections
- **Custom properties** — export and reconcile repository metadata, including multi-select values
- **Strict validation** — unknown and intentionally unsupported YAML keys fail before GitHub is changed
- **Idempotent apply** — a compliant repository produces no mutation requests, including explicit default values that GitHub omits from ruleset responses
- **Automation friendly** — dry-run JSON and distinct success, failure, and drift exit codes
- **Layered policy** — resolve ordered literal configurations with explicit locks and required array elements
- **Semantic policy diff** — compare effective configurations offline with human-readable or JSON output
- **Readable terminal output** — semantic colors distinguish additions, changes, removals, warnings, and errors while honoring `NO_COLOR`
- **Single binary** — calls GitHub directly with no runtime dependency on the GitHub CLI

{: .highlight }
**ghrepocfg** exports or applies settings for one GitHub.com repository per invocation. Resolve and diff operate entirely on local files. Use a shell loop or CI matrix when applying the same configuration to multiple repositories.

## Explore the documentation

Start with [Installation](INSTALL.md), then choose a command:

| Command | What you can do |
|---|---|
| [`export`](commands/export.md) | Capture a repository as portable YAML |
| [`resolve`](commands/resolve.md) | Combine defaults and enforce layered policy |
| [`diff`](commands/diff.md) | Review effective configuration changes offline |
| [`apply`](commands/apply.md) | Preview drift and reconcile a repository |

The [Command Reference](USAGE.md) also covers help, version information, and shared CLI conventions. Learn the schema in the [Configuration Reference](CONFIGURATION.md), design reusable governance with [Layered Policy](POLICY.md), and adapt the [Examples](EXAMPLES.md) to your workflow.
