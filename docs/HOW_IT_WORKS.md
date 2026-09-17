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

No mutation is planned from unreadable state. Readers record unavailable paths separately from values; export and reconciliation consult this metadata. Permission denials and ambiguous resource absence skip affected paths by default. Other read failures abort before the prompt. `--strict` also aborts on incomplete reads.

Collection planning starts with capacity for the desired keys and appends additional live keys as needed. It avoids adding collection lengths when computing allocation sizes, preventing integer overflow in that calculation.

## Idempotence

Comparisons normalize:

- known nil and empty API collections (unreadable collections are never compared);
- GitHub `read`/`write` roles to `pull`/`push`;
- custom repository roles;
- the default ruleset target and bypass mode;
- pending collaborator invitations as existing access;
- variable-name case, label-color case, and SSH public-key comments;
- environment reviewer and branch/tag pattern ordering.

When the repository is compliant, the plan is empty and no mutation request is sent.

## Confirmation and Removals

Human and JSON plans distinguish additions, modifications, and removals. Access and ruleset removals are displayed before the single confirmation prompt. Only `y` or `yes` approves; every other answer rejects the plan.

## Partial Mutation Failures

GitHub does not provide a transaction across repository endpoints. Default execution warns and skips denied writes, while continuing independent changes. Other failures return exit `1`; rate limits stop further writes. Dependent requests are skipped if their prerequisite failed. An error after an earlier step succeeded is a partial failure, never a harmless permission skip. Strict execution stops at the first denied or failed write and verifies managed values after successful execution.

Organization policy, licensing, or permission conflicts therefore do not prevent unrelated independent changes from being attempted.

## Export Behavior

Full export discovers supported configuration APIs and omits unavailable attributes with warnings. The initial repository read is mandatory. Authentication, rate-limit, network, validation, server, and decode errors remain failures. See [Permissions](PERMISSIONS.md).

Scoped export refreshes only existing fields and collections. Unreadable paths retain their previous configured values with an explicit stale-value warning. `--strict` requires complete requested state and leaves the destination unchanged on incomplete reads.

Files are written through a temporary file and atomic rename. Generated key ordering is stable, while comments and custom formatting are not preserved.

## Warnings

Warnings identify important unmanaged state such as legacy protection. They do not weaken strict configuration validation and do not substitute for errors when YAML is invalid.

Verbose apply reports readable repository fields that remain unmanaged. Verbose export reports genuinely unrecognized REST repository response fields to help identify GitHub API evolution.

## Safety Boundaries

The YAML schema cannot express repository visibility, archive state, deletion, transfer, owner, or name. Secrets are excluded because GitHub cannot return secret values for desired-state comparison. Organization settings and inherited rulesets are outside the repository-level boundary.
