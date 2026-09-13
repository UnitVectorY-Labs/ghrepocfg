---
layout: default
title: diff
parent: Command Reference
nav_order: 3
permalink: /commands/diff
---

# ghrepocfg diff
{: .no_toc }

Compare the meaning of two valid literal configurations without contacting GitHub, resolving credentials, selecting a repository, or inspecting Git history.

## Synopsis

```text
ghrepocfg diff [--json] OLD.yaml NEW.yaml
```

## Arguments and options

| Argument or option | Description |
|---|---|
| `OLD.yaml` | Previous literal configuration |
| `NEW.yaml` | New literal configuration |
| `--json` | Emit structured changes instead of human-readable output |
| `-h`, `--help` | Print command usage |

Both paths are required. Place `--json` before the paths. Configuration environment variables and repository discovery are not used.

## Comparison semantics

Both inputs are strictly parsed and validated with the ordinary configuration model. Comments, whitespace, formatting, and YAML key ordering do not produce changes. Unordered collections compare independently of element order; ordered fields, including OIDC claim keys, retain ordering. Domain normalization includes case-insensitive identities and literal defaults.

Presence remains meaningful: an omitted authoritative collection and an explicit empty collection can produce different management behavior and are reported as different. See [semantic comparison](../HOW_IT_WORKS.md#semantic-comparison) for field-specific rules.

Diff compares desired state with desired state. Use [`apply --dry-run`](apply.md) to compare desired state with live repository state.

## Human-readable output

```text
Effective configuration changes:

  /repository/has_wiki
    false -> true

  /repository/topics
    add: ["security-reviewed"]
```

Changes distinguish additions, removals, and modifications using JSON Pointer paths. Unchanged inputs report `No effective configuration changes.` Human output can be captured for a pull request comment; redirected output contains no ANSI styling.

## JSON output

```bash
ghrepocfg diff --json old.yaml new.yaml
```

```json
{
  "changed": true,
  "changes": [
    {
      "path": "/repository/has_wiki",
      "operation": "modify",
      "before": false,
      "after": true
    }
  ]
}
```

`operation` is `add`, `remove`, or `modify`. `before` is present when a previous value exists, and `after` when a new value exists. An explicit JSON `null` is a value, distinct from a missing member. Equal inputs return `{"changed":false,"changes":[]}`.

## Output and exit codes

Human or JSON output goes to stdout; errors go to stderr. Validation errors do not emit a partial diff.

| Code | Meaning |
|---:|---|
| `0` | No effective differences |
| `1` | Invalid arguments, invalid input, or another command error |
| `2` | Effective differences exist |

Use the compiled binary when checking exit codes; `go run` does not preserve its child's exit status. Under `set -e`, capture the status through a conditional:

```bash
status=0
ghrepocfg diff old.yaml new.yaml > policy-diff.txt || status=$?
case "$status" in
  0) echo "Effective policy is unchanged" ;;
  2) cat policy-diff.txt ;;
  *) exit "$status" ;;
esac
```

## Related documentation

Use [`resolve`](resolve.md) to materialize effective configurations and [Layered Policy](../POLICY.md#ci-composition) for CI composition. Revision selection and rollout orchestration belong to the calling automation.
