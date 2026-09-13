---
layout: default
title: version
parent: Command Reference
nav_order: 5
permalink: /commands/version
---

# ghrepocfg version
{: .no_toc }

Print application build and platform information without authentication or network access.

## Synopsis

```text
ghrepocfg version
```

The top-level aliases `ghrepocfg --version` and `ghrepocfg -version` produce the same output.

## Output and exit code

Version information is written to stdout and includes the application version, Go version, operating system, and architecture. The command returns `0`.

Include this output when reporting a problem. See [Installation](../INSTALL.md#verify-the-installation) and [Troubleshooting](../TROUBLESHOOTING.md).
