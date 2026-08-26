---
layout: default
title: ghrepocfg
nav_order: 1
permalink: /
---

# ghrepocfg

Declaratively export, review, and reconcile GitHub repository settings from one portable YAML file.

**ghrepocfg** turns repository governance into a code-reviewable contract. It manages repository behavior, security features, GitHub Actions policy, direct collaborators, team access, and repository rulesets while keeping every proposed change visible before it is applied.

## Key Features

- **Desired-state configuration** — only fields present in YAML are managed
- **Complete drift plans** — inspect every addition, modification, and removal before applying
- **Safe access management** — reconcile direct collaborators, pending invitations, and team permissions authoritatively
- **Repository rulesets** — manage branch, tag, and push rulesets without converting legacy protections
- **Strict validation** — unknown and intentionally unsupported YAML keys fail before GitHub is changed
- **Idempotent apply** — a compliant repository produces no mutation requests
- **Automation friendly** — dry-run JSON and distinct success, failure, and drift exit codes
- **Single binary** — calls GitHub directly with no runtime dependency on the GitHub CLI

## Documentation

- [Installation](INSTALL.md)
- [Usage](USAGE.md)
- [Examples](EXAMPLES.md)
- [Configuration Reference](CONFIGURATION.md)
- [GitHub Features](GITHUB_FEATURES.md)
- [How It Works](HOW_IT_WORKS.md)
- [Troubleshooting](TROUBLESHOOTING.md)

{: .highlight }
**ghrepocfg** manages exactly one GitHub.com repository per invocation. Use a shell loop or CI matrix when applying the same configuration to multiple repositories.
