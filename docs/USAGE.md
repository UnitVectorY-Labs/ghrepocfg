---
layout: default
title: Command Reference
nav_order: 3
has_children: true
permalink: /usage
---

# Command Reference
{: .no_toc }

Choose a command by the work you want to do. Each command page documents its syntax, options, behavior, examples, and exit codes.

| Command | Purpose | GitHub access |
|---|---|---|
| [`export`](commands/export.md) | Capture repository state as literal YAML | Required |
| [`resolve`](commands/resolve.md) | Materialize ordered layers and constraints as literal YAML | None |
| [`diff`](commands/diff.md) | Compare two literal configurations semantically | None |
| [`apply`](commands/apply.md) | Reconcile one repository with one literal configuration | Required |
| [`version`](commands/version.md) | Print build and platform information | None |
| [`help`](commands/help.md) | Find commands and command-specific options | None |

For a first run, start with [Installation](INSTALL.md) and [Examples](EXAMPLES.md). Use the [Configuration Reference](CONFIGURATION.md) for YAML fields and [Layered Policy](POLICY.md) for merge rules and constraints.

## Shared CLI conventions

Options belong after the command name. Place options before positional arguments, as in `ghrepocfg diff --json old.yaml new.yaml`. `resolve` takes its ordered inputs through repeated `--layer` options.

There is no separate validation command: configuration-consuming commands validate automatically. Available flags differ by command; consult its reference page. Repeated verbose flags do not increase verbosity, and there is no quiet flag.

The repository, configuration-path, and authentication environment settings below apply to `export` and `apply`. Offline commands use only their explicit file arguments. `NO_COLOR` applies to human-readable output throughout the CLI.

## Environment Variables

| Variable | Purpose | Precedence |
|---|---|---:|
| `GHREPOCFG_REPO` | Target `OWNER/REPO` when `--repo` is absent | 2 |
| `GHREPOCFG_CONFIG` | Configuration path when `--config` is absent | 2 |
| `GH_TOKEN` | Authentication fallback when GitHub CLI credentials are unavailable | 2 |
| `GITHUB_TOKEN` | Authentication fallback when GitHub CLI credentials and `GH_TOKEN` are unavailable | 3 |
| `NO_COLOR` | Disable ANSI color when set to a non-empty value | — |

Environment variables are not expanded inside YAML. Ordinary configuration files remain literal; layering is available only when explicitly requested with `resolve`, and constraints remain in separate policy files.

## Repository Resolution

For `export` and `apply`, the target repository is resolved in this order:

1. `--repo` or `-R`
2. `GHREPOCFG_REPO`
3. a GitHub.com remote in the current Git repository

`OWNER/REPO`, standard GitHub HTTPS URLs, SCP-style SSH URLs, and `ssh://` GitHub URLs are accepted. When multiple distinct GitHub repositories are configured as remotes, specify `--repo` explicitly.

## Configuration Resolution

For `export` and `apply`, the configuration path is resolved in this order:

1. `--config`
2. `GHREPOCFG_CONFIG`
3. `.ghrepocfg.yaml` at the current Git root

For apply outside a Git checkout, the last fallback is `.ghrepocfg.yaml` in the working directory. Export uses stdout when no appropriate local destination exists.

## Output Streams

- Exported and resolved YAML intended for piping is written to stdout.
- JSON diff and dry-run output is written to stdout without human-readable contamination.
- Warnings, diagnostics, prompts, and errors are written to stderr where appropriate.
- Human-readable plans, effective configuration diffs, and apply success output are written to stdout.

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
| `0` | Successful and compliant, no `diff` changes, or a successful non-dry-run operation |
| `1` | Configuration, authentication, API, cancellation, mutation failure, or `diff` error |
| `2` | Repository drift from `apply --dry-run`, file changes from `export --dry-run`, or effective differences from `diff` |

Use a compiled binary when testing exit codes. The `go run` launcher converts a child exit status such as `2` into its own failure status.

See [Examples](EXAMPLES.md) for complete workflows and [Configuration Reference](CONFIGURATION.md) for every YAML attribute.
