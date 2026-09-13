---
layout: default
title: resolve
parent: Command Reference
nav_order: 2
permalink: /commands/resolve
---

# ghrepocfg resolve
{: .no_toc }

Materialize any ordered sequence of literal configurations and optional policy constraints into one effective configuration. Resolution is entirely local: it requires no GitHub access, authentication, repository selection, or Git history.

## Synopsis

```text
ghrepocfg resolve --layer PATH [--constraints PATH] [--layer PATH ...] [--output PATH]
```

## Options

| Option | Description |
|---|---|
| `--layer PATH` | Add a literal configuration layer; repeat in least-specific to most-specific order |
| `--constraints PATH` | Attach a constraint file to the immediately preceding layer |
| `--output PATH` | Write effective YAML to this file instead of stdout |
| `-h`, `--help` | Print command usage |

At least one layer is required. Each layer may have at most one constraint file. Paths must be supplied explicitly; `GHREPOCFG_CONFIG` and repository discovery are not used.

## Behavior

Every layer must independently be a valid ordinary configuration. Later scalars replace inherited values, objects and maps compose by key, and arrays replace inherited arrays. Omitted values inherit. Explicit empty collections retain their management meaning; an empty map does not clear inherited entries.

After each layer, resolve validates the candidate and checks every inherited constraint before establishing new constraints. Any invalid layer, invalid constraint, or violation fails before a result is emitted. See [Layered Policy](../POLICY.md) for the full merge rules, `exact` and `contains` constraints, and JSON Pointer paths.

Output is canonical ordinary YAML: it normalizes semantic defaults, case-insensitive identities, and unordered collections. Meaningful array ordering and management boundaries are preserved. The result can be passed directly to [`apply`](apply.md).

## Examples

```bash
# Materialize defaults and a repository-specific layer.
ghrepocfg resolve --layer base.yaml --layer repo.yaml > effective.yaml

# Enforce base policy, then add arbitrary intermediate layers.
ghrepocfg resolve \
  --layer base.yaml --constraints base-policy.yaml \
  --layer team.yaml \
  --layer service.yaml \
  --layer repo.yaml \
  --output effective.yaml
```

## Output and exit codes

Without `--output`, YAML goes to stdout. With `--output`, resolve writes the file using a temporary file and atomic rename, leaving an existing destination unchanged if validation fails. Errors go to stderr. Shell redirection (`>`) opens or truncates its destination before the command runs; use `--output` when preserving an existing file on failure matters.

| Code | Meaning |
|---:|---|
| `0` | Effective configuration written successfully |
| `1` | Invalid arguments, input, constraints, policy violation, or output error |

## Related documentation

See [Layered Policy](../POLICY.md) for policy design and CI composition, [`diff`](diff.md) for reviewing effective changes, and the [Configuration Reference](../CONFIGURATION.md) for the literal schema.
