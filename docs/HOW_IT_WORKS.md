---
layout: default
title: How It Works
nav_order: 7
permalink: /how-it-works
---

# How It Works
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Desired State and Scope

YAML presence defines the management boundary. A present scalar is managed, while an omitted scalar is untouched. A present collection is the complete desired collection, while an omitted collection is untouched.

Ordinary export preserves this scope. `export --full` is the explicit operation that expands it.

## Planning Boundary

Apply performs these stages in order:

1. Resolve the repository and configuration path.
2. Strictly decode and validate the complete YAML document.
3. Resolve authentication.
4. Read all state required by every managed section.
5. Normalize GitHub representations and build a structured plan.
6. Render the complete plan.
7. Confirm once unless `--yes` or `--dry-run` was supplied.
8. Execute planned mutations.

No mutation can occur before every managed domain has been read successfully. A permission or API failure during state collection aborts before the prompt.

## Idempotence

Comparisons normalize:

- nil and empty API collections;
- GitHub `read`/`write` roles to `pull`/`push`;
- custom repository roles;
- the default ruleset target and bypass mode;
- pending collaborator invitations as existing access.

When the repository is compliant, the plan is empty and no mutation request is sent.

## Confirmation and Removals

Human and JSON plans distinguish additions, modifications, and removals. Access and ruleset removals are displayed before the single confirmation prompt. Only `y` or `yes` approves; every other answer rejects the plan.

## Partial Mutation Failures

GitHub does not provide a transaction across repository endpoints. After approval, independent mutations continue when one mutation fails. The final result lists successful paths, consolidates every failure, and returns exit code `1` when any requested mutation failed.

Organization policy, licensing, or permission conflicts therefore do not prevent unrelated independent changes from being attempted.

## Export Behavior

Full export reads every supported domain and fails if any authoritative domain cannot be read safely. There is no permission-aware partial export.

Scoped export reads every domain present in the destination and refreshes only its existing fields and collections. It also fails rather than preserving potentially stale or permission-filtered state.

Files are written through a temporary file and atomic rename. Generated key ordering is stable, while comments and custom formatting are not preserved.

## Warnings

Warnings identify important unmanaged state such as legacy protection. They do not weaken strict configuration validation and do not substitute for errors when YAML is invalid.

Verbose apply reports readable repository fields that remain unmanaged. Verbose export reports genuinely unrecognized REST repository response fields to help identify GitHub API evolution.

## Safety Boundaries

The YAML schema cannot express repository visibility, archive state, deletion, transfer, owner, or name. Secrets are excluded because GitHub cannot return secret values for desired-state comparison. Organization settings and inherited rulesets are outside the repository-level boundary.
