---
layout: default
title: GitHub Features
nav_order: 6
permalink: /github-features
---

# GitHub Features
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Supported Features

**ghrepocfg** manages repository-level state on GitHub.com in these domains:

- Repository feature flags, merge behavior, default branch, descriptions, homepage, templates, signoff, forking, and topics
- Repository custom-property values defined by an organization or enterprise
- Dependabot alerts and automated security fixes
- GitHub security-and-analysis feature status when exposed by the repository and plan
- GitHub Actions availability, allowed-action policy, selected-action policy, and default workflow permissions
- Direct collaborators, permissions, and pending invitations
- Organization team access and repository permissions
- Repository-owned branch, tag, and push rulesets

The complete attribute list is in [Configuration Reference](CONFIGURATION.md).

## Repository Access

Direct collaborators and teams are separate authoritative collections. When a collection is present, **ghrepocfg** reads the complete collection before planning additions, permission changes, or removals.

GitHub response roles `read` and `write` are normalized to request permissions `pull` and `push`. Custom repository roles use `custom:ROLE NAME` in YAML.

Pending collaborator invitations are treated as existing access. This prevents a repeated apply from sending the same invitation again and permits authoritative removal of an unwanted pending invitation.

## Custom Properties

Custom-property definitions remain organization- or enterprise-managed. **ghrepocfg** reads and reconciles only the values assigned to the target repository. Each changed value is sent separately so an organization-restricted or invalid value can fail without blocking unrelated property changes. See [Custom Properties](CUSTOM_PROPERTIES.md).

## Repository Rulesets

Rulesets use their names as portable YAML keys. Live numeric IDs are retained only internally for update and deletion requests.

List responses do not include complete rule definitions, so every repository-owned ruleset is fetched individually before comparison. Parent organization and enterprise rulesets are excluded and never proposed for removal.

GitHub returns bypass actors only to callers with sufficient write visibility. Repository admin access is therefore required for authoritative ruleset reads and exports.

## Security Features

Some security endpoints use `404` for both a disabled feature and an inaccessible resource. Repository admin visibility is required before **ghrepocfg** interprets that response as disabled.

GitHub may omit security-and-analysis features that are unavailable because of repository visibility, licensing, product selection, or organization policy. Full export omits unavailable fields instead of guessing their state.

## GitHub Actions

Actions policy, selected-action policy, and workflow permissions use separate GitHub endpoints. Selected-action details can be read only while the live policy is `selected`; GitHub returns `409 Conflict` otherwise.

## Legacy Protections

Legacy branch and tag protections are not managed or converted. Detection is best-effort:

- protected branches are checked through the legacy branch protection endpoint;
- the retired legacy tag protection endpoint is queried when available;
- detected legacy protection produces a warning.

A generated configuration should not be assumed to represent legacy protection shown in a warning.

## Intentionally Unsupported

- Repository visibility and public/private/internal transitions
- Repository archive state, deletion, transfer, name, and owner
- Repository secrets or other write-only values
- Organization and enterprise configuration, including custom-property definitions and inherited rulesets
- Legacy branch and tag protection mutation or migration
- GitHub Enterprise Server and custom API base URLs
- Deploy keys, webhooks, environments, Pages, issue labels, variables, and unlisted repository collections
- Permission-aware partial export
- Multi-repository orchestration inside the application

Unsupported high-risk and secret keys are rejected with actionable validation errors rather than silently accepted.

## API Behavior

The application calls `https://api.github.com` directly, requests `application/vnd.github+json`, and sends the stable `2022-11-28` API version header. Pagination uses GitHub `Link` headers and requests up to 100 entries per page.

Verbose export recognizes normal read-only repository metadata and reports only REST response fields unknown to the installed version. Unknown response fields remain invalid YAML keys until explicitly supported by a release.
