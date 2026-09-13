---
layout: default
title: help
parent: Command Reference
nav_order: 6
permalink: /commands/help
---

# ghrepocfg help
{: .no_toc }

Display available commands or the options for a specific command. Help requires no credentials or network access.

## Synopsis

```text
ghrepocfg help
ghrepocfg COMMAND --help
```

## Examples

```bash
ghrepocfg help
ghrepocfg resolve --help
ghrepocfg diff --help
```

`ghrepocfg --help` and `ghrepocfg -h` are aliases for top-level help. Commands that accept flags also support `-h`. Use `ghrepocfg COMMAND --help` for command-specific usage; `ghrepocfg help COMMAND` does not select that command's help.

## Output and exit codes

Top-level help is written to stdout. Command-specific flag help is written to stderr. Both return `0`. Invoking `ghrepocfg` without a command prints usage to stderr and returns `1`.

See the [Command Reference](../USAGE.md) for shared CLI conventions and links to every command.
