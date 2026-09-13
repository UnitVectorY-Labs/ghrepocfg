---
layout: default
title: apply
parent: Command Reference
nav_order: 4
permalink: /commands/apply
---

# ghrepocfg apply
{: .no_toc }

Reconcile one GitHub.com repository with one literal desired-state configuration.

## Synopsis

```text
ghrepocfg apply [--repo OWNER/REPO] [--config PATH] [--dry-run] [--json] [--yes] [--verbose]
```

## Options

| Option | Description |
|---|---|
| `-R`, `--repo OWNER/REPO` | Target repository; see [repository resolution](../USAGE.md#repository-resolution) for defaults |
| `--config PATH` | Literal YAML input; see [configuration resolution](../USAGE.md#configuration-resolution) for defaults |
| `--dry-run` | Read and plan without prompting or changing GitHub |
| `--json` | Emit a structured plan; requires `--dry-run` |
| `-y`, `--yes` | Approve all planned changes without prompting |
| `-v`, `--verbose` | Emit additional diagnostics |
| `-h`, `--help` | Print command usage |

## Behavior

Apply strictly validates the configuration, resolves [authentication](../INSTALL.md#authentication), reads all managed state, and builds one complete plan before making changes. A read failure aborts before any mutation or confirmation prompt.

With no drift, apply reports `No changes.` and sends no mutation requests. Otherwise it prompts once for the complete plan; the default answer is no. `--yes` approves all changes, while `--dry-run` never prompts or mutates.

After approval, independent mutations continue if one fails. Successful paths and all failures are reported at the end. GitHub does not provide a transaction across these operations.

Apply consumes one literal configuration. It does not interpret policy layers or constraint files. Materialize those with [`resolve`](resolve.md) first. See the [Configuration Reference](../CONFIGURATION.md) for management boundaries and supported fields.

## Examples

```bash
# Inspect repository drift.
ghrepocfg apply --repo acme/service --config .ghrepocfg.yaml --dry-run

# Capture a structured drift plan.
ghrepocfg apply --repo acme/service --config .ghrepocfg.yaml --dry-run --json

# Reconcile a materialized policy without an interactive prompt.
ghrepocfg apply --repo acme/service --config effective.yaml --yes
```

## Output and exit codes

Human-readable plans and successful changes go to stdout; prompts, errors, and diagnostics go to stderr. JSON plans contain `repository`, `drift`, and `changes`, with `warnings` and `unmanaged` included when applicable. Changes identify their `operation`, `path`, and applicable `before` and `after` values. A replacement operation denotes deletion followed by creation.

| Code | Meaning |
|---:|---|
| `0` | Repository was compliant, or all planned changes succeeded |
| `1` | Invalid input, authentication or API error, cancellation, or mutation failure |
| `2` | A dry-run found repository drift |

## Related documentation

See [How It Works](../HOW_IT_WORKS.md) for planning and execution, [Examples](../EXAMPLES.md) for workflows, and [`diff`](diff.md) for comparing local desired states without reading GitHub.
