---
layout: default
title: Installation
nav_order: 2
permalink: /install
---

# Installation
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Prerequisites

- **GitHub.com repository:** GitHub Enterprise Server is not supported in v1
- **GitHub authentication:** an authenticated GitHub CLI session, `GH_TOKEN`, or `GITHUB_TOKEN`
- **Latest version of Go:** required only for `go install` or building from source

The installed binary calls GitHub directly. The `gh` executable is an optional credential source and is not needed for normal API operations.

## Installation Methods

### Download Binary

Download a pre-built binary from [GitHub Releases](https://github.com/UnitVectorY-Labs/ghrepocfg/releases).

[![GitHub release](https://img.shields.io/github/release/UnitVectorY-Labs/ghrepocfg.svg)](https://github.com/UnitVectorY-Labs/ghrepocfg/releases/latest)

Choose the binary for your platform, make it executable where necessary, and place it on your `PATH`.

### Install Using Go

```bash
go install github.com/UnitVectorY-Labs/ghrepocfg@latest
```

Ensure the Go binary directory is on your `PATH`.

### Build from Source

```bash
git clone https://github.com/UnitVectorY-Labs/ghrepocfg.git
cd ghrepocfg
go build -o ghrepocfg
```

## Verify the Installation

```bash
ghrepocfg version
```

Version output includes the application version, Go version, operating system, and architecture.

## Authentication

Credentials are resolved in this order:

1. `gh auth token --hostname github.com`, when the GitHub CLI is installed and authenticated
2. `GH_TOKEN`
3. `GITHUB_TOKEN`

Authenticate the GitHub CLI with:

```bash
gh auth login
```

Or provide a token to the process environment. Tokens are sent only to `https://api.github.com`, are never written to YAML, and are not included in API errors.

{: .highlight }
A usable GitHub CLI credential takes precedence over token environment variables. Run without `gh` on `PATH` when an automation environment must use `GH_TOKEN` or `GITHUB_TOKEN` instead.

## GitHub Permissions

Use the least privilege that covers every configured section.

| Feature | Read or dry-run | Apply |
|---|---|---|
| Repository settings and topics | Metadata read | Administration write |
| Security settings | Repository admin visibility | Administration or applicable security-feature write access |
| Actions policy | Actions policy read access | Administration or Actions policy write access |
| Direct collaborators and invitations | Repository administration read | Administration write |
| Team access | Repository administration and organization Members read | Administration write and Members read |
| Repository rulesets | Repository administration read | Administration write |

Full export and authoritative security, collaborator, team, or ruleset management require repository admin access. This prevents GitHub permission filtering from being mistaken for an empty desired collection.

Classic personal access tokens generally need `repo` for private repositories and organization scopes for team visibility. Fine-grained token permission names and feature licensing are enforced by GitHub.
