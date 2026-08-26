---
layout: default
title: Usage
nav_order: 3
permalink: /usage
---

# Usage
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Commands

```text
ghrepocfg export [flags]
ghrepocfg apply [flags]
ghrepocfg version
```

There is no separate validate, diff, or check command. Configuration validation is automatic, `apply --dry-run` checks repository drift, and `export --dry-run` previews configuration-file changes.

## Export

```text
ghrepocfg export [--repo OWNER/REPO] [--config PATH] [--full] [--dry-run]
```

`export` reads supported repository state and produces YAML.

- A new destination receives every supported setting that can be safely read.
- An existing destination preserves its management scope and refreshes only fields and collections already present.
- `--full` replaces an existing management scope with a complete export.
- `--dry-run` compares against a file destination, prints what would change, writes nothing, and never prompts.
- Without a file destination, YAML is written to stdout and diagnostics remain on stderr.

When no `--config` is supplied, export writes `.ghrepocfg.yaml` at the Git root only when the local checkout corresponds to the target repository. Otherwise it writes YAML to stdout.

## Apply

```text
ghrepocfg apply [--repo OWNER/REPO] [--config PATH] [--dry-run] [-y]
```

`apply` strictly validates YAML, reads all managed state, builds one complete plan, and displays that plan before changing anything.

- With no drift, it reports `No changes.` and makes no mutation requests.
- By default, one confirmation prompt covers the entire plan and defaults to no.
- `-y` or `--yes` approves the plan without prompting.
- `--dry-run` never prompts or mutates and returns exit code `2` when drift exists.
- `--json` emits a structured plan when combined with `--dry-run`.

After approval, independent mutations continue if one mutation fails. Successful paths and every failure are reported at the end.

## Flags

| Flag | Commands | Description |
|---|---|---|
| `-R`, `--repo OWNER/REPO` | export, apply | Target repository |
| `--config PATH` | export, apply | YAML source or destination |
| `--full` | export | Replace the existing management scope with all safely readable supported settings |
| `--dry-run` | export, apply | Preview without writing or prompting |
| `-y`, `--yes` | apply | Skip the confirmation prompt |
| `--json` | export, apply | Emit structured dry-run output |
| `-v`, `--verbose` | export, apply | Emit one level of additional diagnostics |
| `-h`, `--help` | export, apply | Print command usage |

Repeated verbose flags do not create additional verbosity levels. There is no quiet flag.

## Environment Variables

| Variable | Purpose | Precedence |
|---|---|---:|
| `GHREPOCFG_REPO` | Target `OWNER/REPO` when `--repo` is absent | 2 |
| `GHREPOCFG_CONFIG` | Configuration path when `--config` is absent | 2 |
| `GH_TOKEN` | Authentication fallback when GitHub CLI credentials are unavailable | 2 |
| `GITHUB_TOKEN` | Authentication fallback when GitHub CLI credentials and `GH_TOKEN` are unavailable | 3 |
| `NO_COLOR` | Disable ANSI color when set to a non-empty value | — |

Environment variables are not expanded inside YAML. Configuration is literal: no templates, includes, inheritance, layering, substitution, or expression evaluation are supported.

## Repository Resolution

The target repository is resolved in this order:

1. `--repo` or `-R`
2. `GHREPOCFG_REPO`
3. a GitHub.com remote in the current Git repository

`OWNER/REPO`, standard GitHub HTTPS URLs, SCP-style SSH URLs, and `ssh://` GitHub URLs are accepted. When multiple distinct GitHub repositories are configured as remotes, specify `--repo` explicitly.

## Configuration Resolution

The path is resolved in this order:

1. `--config`
2. `GHREPOCFG_CONFIG`
3. `.ghrepocfg.yaml` at the current Git root

For apply outside a Git checkout, the last fallback is `.ghrepocfg.yaml` in the working directory. Export uses stdout when no appropriate local destination exists.

## Output Streams

- Exported YAML intended for piping is written to stdout.
- JSON dry-run output is written to stdout without human-readable contamination.
- Warnings, diagnostics, prompts, and errors are written to stderr where appropriate.
- Human-readable plans and apply success output are written to stdout.

## Color Output

Interactive terminal output uses color to distinguish meaning:

- cyan for repositories and setting paths;
- green for additions, desired values, successful operations, and compliant results;
- yellow for warnings and previous values;
- red for removals, failures, and errors;
- dim text for arrows, prompts, verbose context, and unmanaged settings.

Color is enabled only when the corresponding output stream is an interactive terminal. Redirected and piped output remains plain. Set `NO_COLOR` to any non-empty value to disable ANSI color, following the [`NO_COLOR` convention](https://no-color.org/):

```bash
NO_COLOR=1 ghrepocfg apply --dry-run
```

`TERM=dumb` also disables styling. YAML and JSON output never contains ANSI escape sequences.

## Exit Codes

| Code | Meaning |
|---:|---|
| `0` | Successful and compliant, or a successful non-dry-run operation |
| `1` | Configuration, authentication, API, cancellation, or mutation failure |
| `2` | Repository drift from `apply --dry-run`, or file changes from `export --dry-run` |

Use a compiled binary when testing exit codes. The `go run` launcher converts a child exit status such as `2` into its own failure status.

See [Examples](EXAMPLES.md) for complete workflows and [Configuration Reference](CONFIGURATION.md) for every YAML attribute.
