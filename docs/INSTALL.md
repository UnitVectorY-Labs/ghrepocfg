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

## Permissions

Grant only the permissions needed by the fields you manage. Fine-grained PATs and GitHub Apps use endpoint-specific repository permissions; account roles and organization policy still apply. Custom-property reads need Metadata read; writes need Custom properties write. Most Actions policy settings need Administration, while OIDC needs Actions and repository variables need Variables.

See [Permissions and partial access](PERMISSIONS.md) for the complete read/write matrix, classic-token scopes, and `--strict`. Default operation warns and skips inaccessible attributes; strict operation requires complete requested state.
