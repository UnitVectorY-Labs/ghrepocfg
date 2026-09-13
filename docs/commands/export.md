---
layout: default
title: export
parent: Command Reference
nav_order: 1
permalink: /commands/export
---

# ghrepocfg export
{: .no_toc }

Capture supported settings from one GitHub.com repository as an ordinary literal configuration.

## Synopsis

```text
ghrepocfg export [--repo OWNER/REPO] [--config PATH] [--full] [--dry-run] [--json] [--verbose]
```

## Options

| Option | Description |
|---|---|
| `-R`, `--repo OWNER/REPO` | Target repository; see [repository resolution](../USAGE.md#repository-resolution) for defaults |
| `--config PATH` | Destination YAML file; defaults are described below |
| `--full` | Replace the destination's management scope with all supported state that can safely be read |
| `--dry-run` | Preview changes to a file destination without writing or prompting |
| `--json` | Emit structured changes; requires `--dry-run` |
| `-v`, `--verbose` | Emit additional diagnostics |
| `-h`, `--help` | Print command usage |

## Behavior and destination

A new destination receives all supported settings that can safely be read. When a destination already exists, export refreshes only its managed fields and collections. Use `--full` to replace that scope. Omitted fields remain unmanaged; exported collections remain authoritative.

The destination is selected from `--config`, then `GHREPOCFG_CONFIG`. Without either, export writes `.ghrepocfg.yaml` at the Git root only when the checkout corresponds to the target repository. Otherwise it writes YAML to stdout. File writes use a temporary file and atomic rename; comments and custom formatting are not preserved.

`--dry-run` requires a file destination. It prints additions, removals, and modifications without changing the file. JSON output contains `repository`, `changed`, `changes`, and `warnings`; change entries contain `operation`, `path`, and applicable `before` and `after` values.

Export requires [authentication](../INSTALL.md#authentication). Full export may omit unavailable feature groups with warnings; explicitly managed groups must be readable. See [GitHub Features](../GITHUB_FEATURES.md).

## Examples

```bash
# Capture a repository into a new file, or refresh an existing file's scope.
ghrepocfg export --repo acme/service --config .ghrepocfg.yaml

# Preview a complete replacement of the exported scope.
ghrepocfg export --repo acme/service --config .ghrepocfg.yaml --full --dry-run

# Capture a machine-readable preview.
ghrepocfg export --repo acme/service --config .ghrepocfg.yaml --dry-run --json
```

## Output and exit codes

YAML and previews go to stdout. File-write confirmations, warnings, and diagnostics go to stderr. JSON and YAML contain no terminal styling.

| Code | Meaning |
|---:|---|
| `0` | Export succeeded, or a dry-run found no file changes |
| `1` | Invalid input, authentication, API, or file error |
| `2` | A dry-run found file changes |

## Related documentation

Use [`apply`](apply.md) to reconcile the exported configuration, [`resolve`](resolve.md) to layer it, or [`diff`](diff.md) to compare it with another file. See [How It Works](../HOW_IT_WORKS.md#export-behavior) for scope-preserving export details.
