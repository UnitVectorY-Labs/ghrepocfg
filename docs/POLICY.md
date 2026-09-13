---
layout: default
title: Layered Policy
nav_order: 6
permalink: /policy
---

# Layered Policy
{: .no_toc }

`ghrepocfg` configuration files are literal desired state. The `resolve` command materializes an ordered stack of those ordinary files into one effective file. The result is a regular configuration that can be passed directly to `apply`.

Resolution is local and deterministic. It does not inspect Git, select repositories, call GitHub, or require authentication. Any meaning assigned to layers belongs to the calling script.

## Resolve layers

Layers are supplied from least specific to most specific. Repeat `--layer` for any number of files:

```bash
ghrepocfg resolve \
  --layer enterprise.yaml \
  --layer payments.yaml \
  --layer service.yaml \
  --layer repository.yaml > effective.yaml
```

Use `--output PATH` to write the result to a file. Without it, YAML goes to stdout.

Every layer must independently be a valid `ghrepocfg` configuration. Layers are not fragments wrapped in `values:` and do not change the ordinary schema. Unknown keys, invalid values, malformed YAML, and a merged result that fails validation are errors before output is produced.

## Merge semantics

* Omitted values inherit the value already resolved.
* A scalar in a later layer replaces the inherited scalar, including `false` and an empty string where valid.
* Objects and maps compose by key. A child can replace an inherited entry; an empty map does not clear inherited entries.
* Arrays replace the inherited array as a whole. An explicit `[]` clears an inherited authoritative collection; omission leaves it inherited.

These rules keep omission distinct from an explicit empty collection. `null` follows ordinary parser semantics for the field; custom-property values may use `null` as their literal unset value.

## Constraint files

A constraint file is separate from a layer and is attached to the layer specified by the same `--layer` occurrence. Supply it with `--constraints`:

```bash
ghrepocfg resolve \
  --layer base.yaml --constraints base-policy.yaml \
  --layer team.yaml \
  --layer repo.yaml
```

Constraints accumulate as resolution proceeds. An earlier constraint governs every more-specific layer. A later layer can add constraints, but cannot weaken, remove, or bypass an inherited one.

```yaml
locks:
  - path: /actions/default_workflow_permissions
    mode: exact
  - path: /repository/topics
    mode: contains
```

Paths are JSON Pointers. The empty pointer (`""`) names the configuration root. Escape `~` as `~0` and `/` as `~1`; array path segments are canonical, zero-based indices. A pointer to a nullable pointer field that is written as `null` is treated as omitted and inherits; non-pointer `null` follows the field's ordinary literal zero-value semantics.

`exact` locks the complete semantic value at the path after its layer is applied. It works for scalars, objects, maps, and arrays; replacing a parent object cannot bypass a locked descendant. `contains` applies only to arrays. The array value established by the declaring layer becomes mandatory, while later layers may add elements. Membership ignores relative ordering, including for an otherwise ordered array. Use `exact` on the whole array when order must be protected. A later `contains` constraint cannot downgrade or weaken an inherited `exact` lock.

Required elements use the field's semantic equality, supporting scalar arrays such as topics and structured arrays such as reviewers. Unordered arrays are canonically sorted; ordered arrays retain their order and normalization rules. Map identities and case rules are likewise canonicalized before constraint checks.

The constrained path must exist in the typed effective configuration after the declaring layer, either inherited or supplied by that layer. Literal model defaults, such as an omitted ruleset target meaning `branch`, are effective values too. A missing path, malformed JSON Pointer, wrong mode for the value type, contradictory constraint, or attempted violation fails locally with the path and layers involved identified in the diagnostic.

For example, this base policy prevents a repository layer from changing workflow permissions:

```yaml
# base.yaml
actions:
  default_workflow_permissions: read
```

```yaml
# base-policy.yaml
locks:
  - path: /actions/default_workflow_permissions
    mode: exact
```

This override is invalid:

```yaml
actions:
  default_workflow_permissions: write
```

Required topics can still be extended:

```yaml
# governance.yaml
repository:
  topics: [security-reviewed, centrally-managed]
```

```yaml
# governance-policy.yaml
locks:
  - path: /repository/topics
    mode: contains
```

`[security-reviewed, centrally-managed, payments]` is valid; `[payments]` is not.

## Reviewing effective policy

Use [`diff`](commands/diff.md) to compare two materialized configurations before reconciling a repository. Its command reference describes human and JSON output, semantic comparison, and exit codes. The [`resolve` reference](commands/resolve.md) documents layer options and output-file behavior.

## CI composition

```bash
ghrepocfg resolve --layer base.yaml --layer repo.yaml > new.yaml
set +e
ghrepocfg diff old.yaml new.yaml
status=$?
set -e
case "$status" in
  0) echo "Effective policy is unchanged" ;;
  1) echo "Could not compare policy" >&2; exit 1 ;;
  2)
  ghrepocfg apply --repo OWNER/REPO --config new.yaml --yes
  ;;
esac
```

Revision selection, repository inventory, matrices, concurrency, and rollout policy remain the responsibility of surrounding CI or shell scripts.
