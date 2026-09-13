---
layout: default
title: How It Works
nav_order: 8
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

## Offline Policy Pipeline

The commands form a deliberately small pipeline. `export` reads GitHub state and writes one literal configuration. `resolve` accepts any ordered set of literal layers and their separate constraints, then emits one literal effective configuration. `diff` compares two literal configurations locally. `apply` consumes one literal configuration and reconciles one selected repository.

```text
export (GitHub state) -> literal config
literal layers + constraints -> resolve -> literal effective config
literal config <-> literal config -> diff
literal effective config -> apply (one repository)
```

`resolve` and `diff` do not use Git history, repository selection, GitHub APIs, or authentication. Revision selection, inventory, and orchestration stay in the surrounding shell or CI system.

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
- pending collaborator invitations as existing access;
- variable-name case, label-color case, and SSH public-key comments;
- environment reviewer and branch/tag pattern ordering.

When the repository is compliant, the plan is empty and no mutation request is sent.

## Semantic Comparison

The same semantic representation is used by resolution constraints and offline diffing. Unordered arrays include repository topics, custom-property multi-select values, selected-action patterns, CodeQL languages, environment reviewers and branch/tag patterns, delegated-bypass reviewers, ruleset rules and bypass actors, ruleset rule selections, and ref-name include/exclude selections. OIDC claim keys remain ordered. Ordered arrays change when reordered.

| Value or field | Semantic comparison |
|---|---|
| Repository topics | Unordered; canonical sorted order |
| Custom-property multi-select values | Unordered selection |
| Selected-action patterns; CodeQL languages | Unordered |
| Environment reviewers and branch/tag patterns | Unordered |
| Ruleset rules, bypass actors, and rule selections | Unordered structured values |
| OIDC `include_claim_keys` | Ordered; reordering changes meaning |
| Collaborator, team, label, and environment map names | Case-insensitive identity |
| Action and environment variable names | Case-insensitive identity, canonical casing |
| Label colors; SSH public keys | Color case ignored; SSH comments ignored |
| Ruleset target and bypass mode | Default values materialized before comparison |

Identity and normalization rules also apply consistently: collaborator, team, label, and environment names are case-insensitive; action and environment variable names are canonicalized case-insensitively; label colors compare case-insensitively; SSH key comments are ignored; and ruleset default target and bypass mode values are materialized before comparison.

Resolved output is canonical: map keys and unordered arrays are emitted in stable order. JSON Pointer paths use `~0` for `~` and `~1` for `/`; the empty pointer (`""`) names the configuration root. Array indices address the canonical semantic array order. For unordered arrays, prefer locking the whole array with `exact`; `contains` checks membership and intentionally ignores relative ordering even when the underlying field is ordered.

## Confirmation and Removals

Human and JSON plans distinguish additions, modifications, and removals. Access and ruleset removals are displayed before the single confirmation prompt. Only `y` or `yes` approves; every other answer rejects the plan.

## Partial Mutation Failures

GitHub does not provide a transaction across repository endpoints. After approval, independent mutations continue when one mutation fails. The final result lists successful paths, consolidates every failure, and returns exit code `1` when any requested mutation failed.

Organization policy, licensing, or permission conflicts therefore do not prevent unrelated independent changes from being attempted.

## Export Behavior

Full export reads the core domains and discovers additional configuration APIs. Unavailable additional groups are omitted with warnings; failures in core domains and unexpected errors remain fatal. Explicitly managed groups always require successful reads. See [GitHub Features](GITHUB_FEATURES.md#additional-configuration-apis) for the availability policy.

Scoped export reads every domain present in the destination and refreshes only its existing fields and collections. It also fails rather than preserving potentially stale or permission-filtered state.

Files are written through a temporary file and atomic rename. Generated key ordering is stable, while comments and custom formatting are not preserved.

## Warnings

Warnings identify important unmanaged state such as legacy protection. They do not weaken strict configuration validation and do not substitute for errors when YAML is invalid.

Verbose apply reports readable repository fields that remain unmanaged. Verbose export reports genuinely unrecognized REST repository response fields to help identify GitHub API evolution.

## Safety Boundaries

The YAML schema cannot express repository visibility, archive state, deletion, transfer, owner, or name. Secrets are excluded because GitHub cannot return secret values for desired-state comparison. Organization settings and inherited rulesets are outside the repository-level boundary.
