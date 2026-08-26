---
layout: default
title: Examples
nav_order: 4
permalink: /examples
---

# Examples
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Capture a Baseline

From a local checkout with a GitHub remote:

```bash
cd my-reference-repository
ghrepocfg export
git add .ghrepocfg.yaml
```

To export a repository without using local inference:

```bash
ghrepocfg export --repo acme/reference --config reference.yaml
```

To pipe YAML:

```bash
ghrepocfg export --repo acme/reference > reference.yaml
```

## Manage Selected Repository Settings

Omit everything that should remain unmanaged:

```yaml
repository:
  has_issues: true
  has_wiki: false
  allow_squash_merge: true
  allow_merge_commit: false
  delete_branch_on_merge: true
```

`has_discussions`, topics, access, security, Actions, and rulesets remain untouched because they are absent.

## Manage Topics Authoritatively

```yaml
repository:
  topics:
    - go
    - github-api
    - governance
```

The list is the complete desired topic set. Use `topics: []` to remove all topics.

## Manage Repository Access

```yaml
collaborators:
  octocat:
    permission: push
  monalisa:
    permission: pull

teams:
  platform:
    permission: maintain
  security:
    permission: admin
```

Because both sections are present, they are authoritative. Extra direct collaborators, pending invitations, and repository team associations are proposed for removal. Omit a whole section to leave that access type unmanaged.

## Remove Every Direct Collaborator

```yaml
collaborators: {}
```

An empty present collection means “manage this collection as empty.” It is different from omitting `collaborators`.

## Manage GitHub Actions Policy

```yaml
actions:
  enabled: true
  allowed_actions: selected
  selected_actions:
    github_owned_allowed: true
    verified_allowed: true
    patterns_allowed:
      - actions/cache@*
      - docker/*
  default_workflow_permissions: read
  can_approve_pull_request_reviews: false
```

GitHub exposes selected-action details only while selected actions are active. Establish `allowed_actions: selected` before introducing selected patterns when migrating from another policy.

## Manage Security Features

```yaml
security:
  vulnerability_alerts: true
  automated_security_fixes: true
  secret_scanning:
    status: enabled
  secret_scanning_push_protection:
    status: enabled
```

Feature availability depends on repository visibility, organization policy, licensing, and token permissions. Full export omits security fields that GitHub does not expose.

## Protect the Default Branch with a Ruleset

```yaml
rulesets:
  protect-default:
    target: branch
    enforcement: active
    bypass_actors: []
    conditions:
      ref_name:
        include:
          - "~DEFAULT_BRANCH"
        exclude: []
    rules:
      - type: deletion
      - type: non_fast_forward
      - type: pull_request
        parameters:
          allowed_merge_methods:
            - squash
            - merge
          dismiss_stale_reviews_on_push: true
          require_code_owner_review: true
          require_last_push_approval: true
          required_approving_review_count: 1
          required_review_thread_resolution: true
```

Rulesets are keyed by name so the same file can be reused across repositories. Organization and enterprise rulesets are never added to or removed from this collection.

## Preview Repository Drift

```bash
ghrepocfg apply \
  --repo acme/service \
  --config reference.yaml \
  --dry-run
```

Example output:

```text
Repository: acme/service

Changes:

  repository.has_wiki
    true -> false

  teams.platform.permission
    "push" -> "maintain"

  collaborators.former-user
    remove: "pull"
```

No confirmation or mutation occurs. Exit code `2` indicates drift.

## Use JSON in CI

```bash
ghrepocfg apply --dry-run --json > ghrepocfg-plan.json
status=$?

case "$status" in
  0) echo "Repository is compliant" ;;
  2) echo "Repository drift detected" >&2; exit 2 ;;
  *) echo "ghrepocfg failed" >&2; exit "$status" ;;
esac
```

## Apply Non-Interactively

```bash
ghrepocfg apply \
  --repo acme/service \
  --config reference.yaml \
  --yes
```

`--yes` skips only the prompt. Validation, complete state reads, planning, mutation failure aggregation, and exit codes remain unchanged.

## Refresh an Existing Configuration

```bash
ghrepocfg export --config .ghrepocfg.yaml
```

Only already-present fields and collections are refreshed. To preview the file changes:

```bash
ghrepocfg export --config .ghrepocfg.yaml --dry-run
```

To intentionally expand the file to the full supported scope:

```bash
ghrepocfg export --config .ghrepocfg.yaml --full
```

## Apply One Configuration to Several Repositories

```bash
for repo in api worker web; do
  ghrepocfg apply \
    --repo "acme/$repo" \
    --config .ghrepocfg.yaml \
    --yes || exit
done
```

Repository selection, ordering, concurrency, and stop/continue policy remain explicit in the calling shell or CI matrix.
