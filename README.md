[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Active](https://img.shields.io/badge/Status-Active-green)](https://guide.unitvectorylabs.com/bestpractices/status/#active) 

# ghrepocfg

Declaratively manage, review, export, layer, compare, and synchronize GitHub repository settings from configuration files.

`ghrepocfg` captures supported repository settings as literal YAML, previews drift as a complete plan, and applies only what changed. It handles repository settings, custom properties, security controls, GitHub Actions policy, direct collaborators, team access, repository rulesets, deployment environments, Pages, labels, autolinks, and deploy keys in one standalone Go binary.

Build reusable governance with `resolve`, which combines ordered literal configurations and optional policy constraints into one effective file. Review policy changes with the offline, semantic `diff` command before deciding whether to apply them.

```console
$ ghrepocfg export --repo acme/reference --config .ghrepocfg.yaml
Exported acme/reference to .ghrepocfg.yaml

$ ghrepocfg apply --repo acme/service --dry-run
Repository: acme/service

Changes:

  repository.has_wiki
    true -> false
```

Configuration is safe by construction: unknown YAML keys fail, omitted scalar fields are untouched, managed collections are authoritative, visibility and archive state cannot be changed, and every mutation waits until the complete state has been read and planned. Interactive output uses semantic color for fast scanning and honors [`NO_COLOR`](https://no-color.org/).

## Install

Download a binary from [GitHub Releases](https://github.com/UnitVectorY-Labs/ghrepocfg/releases), or build from source with a current Go toolchain:

```bash
go install github.com/UnitVectorY-Labs/ghrepocfg@latest
```

The binary calls GitHub directly. The GitHub CLI is optional; when installed and authenticated, its token is used automatically.

## Start in a repository

```bash
cd my-repository
ghrepocfg export
git add .ghrepocfg.yaml

# Exit 0 means compliant; exit 2 means drift.
ghrepocfg apply --dry-run

# Review one complete plan and confirm once.
ghrepocfg apply
```

Use `--repo owner/repo` and `--config PATH` to apply one portable configuration elsewhere. Add `--yes` for CI or other non-interactive automation.

## Documentation

- [Installation](docs/INSTALL.md)
- [Command reference](docs/USAGE.md): [export](docs/commands/export.md), [resolve](docs/commands/resolve.md), [diff](docs/commands/diff.md), [apply](docs/commands/apply.md), [version](docs/commands/version.md), [help](docs/commands/help.md)
- [Examples](docs/EXAMPLES.md)
- [Configuration reference](docs/CONFIGURATION.md)
- [Layered policy](docs/POLICY.md)
- [GitHub features](docs/GITHUB_FEATURES.md)
- [How it works](docs/HOW_IT_WORKS.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

`export` and `apply` target one GitHub.com repository per invocation; `resolve` and `diff` operate on local files. Multi-repository orchestration belongs in the shell, where the selection and rollout remain explicit.
