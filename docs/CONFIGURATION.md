---
layout: default
title: Configuration Reference
nav_order: 5
permalink: /configuration
---

# Configuration Reference
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## File Format

The default file is `.ghrepocfg.yaml` at the repository root. v1 accepts one YAML document only. JSON configuration, templates, includes, inheritance, variable substitution, environment interpolation, configuration layering, and expressions are not supported.

Unknown keys at every modeled level are errors. Empty strings, `false`, `0`, `[]`, and `{}` are literal desired values.

## Management Semantics

- A present scalar or object field is managed.
- An omitted scalar or object field is unmanaged and remains unchanged.
- A present collection is authoritative, including an empty collection.
- An omitted collection is entirely unmanaged.

## Top-Level Sections

| Key | Shape | Semantics |
|---|---|---|
| `repository` | object | Each present field is managed independently |
| `custom_properties` | map keyed by property name | Present map is authoritative |
| `security` | object | Each present feature is managed independently |
| `actions` | object | Each present field is managed independently |
| `collaborators` | map keyed by GitHub login | Present map is authoritative |
| `teams` | map keyed by organization team slug | Present map is authoritative |
| `rulesets` | map keyed by unique ruleset name | Present map is authoritative for repository-owned rulesets |

## Repository

| Key | Type or Values | Description |
|---|---|---|
| `description` | string | Repository description; `""` clears it |
| `homepage` | string | Repository homepage; `""` clears it |
| `has_issues` | boolean | Enable issues |
| `has_projects` | boolean | Enable repository projects |
| `has_wiki` | boolean | Enable the wiki |
| `has_discussions` | boolean | Enable discussions |
| `has_pull_requests` | boolean | Allow pull requests, primarily for forks |
| `pull_request_creation_policy` | `all`, `collaborators_only` | Who may create pull requests |
| `is_template` | boolean | Make the repository available as a template |
| `default_branch` | string | Existing branch to use as the default |
| `allow_squash_merge` | boolean | Allow squash merges |
| `allow_merge_commit` | boolean | Allow merge commits |
| `allow_rebase_merge` | boolean | Allow rebase merges |
| `allow_auto_merge` | boolean | Allow pull request auto-merge |
| `delete_branch_on_merge` | boolean | Delete head branches after merge |
| `allow_update_branch` | boolean | Allow an out-of-date pull request branch to be updated |
| `use_squash_pr_title_as_default` | boolean | Legacy GitHub preference retained for repositories that return it |
| `squash_merge_commit_title` | `PR_TITLE`, `COMMIT_OR_PR_TITLE` | Default squash title |
| `squash_merge_commit_message` | `PR_BODY`, `COMMIT_MESSAGES`, `BLANK` | Default squash message |
| `merge_commit_title` | `PR_TITLE`, `MERGE_MESSAGE` | Default merge-commit title |
| `merge_commit_message` | `PR_TITLE`, `PR_BODY`, `BLANK` | Default merge-commit message |
| `web_commit_signoff_required` | boolean | Require signoff for web commits |
| `allow_forking` | boolean | Allow private-repository forking |
| `topics` | array of strings | Complete desired topic set |

Repository `name`, `owner`, `private`, `visibility`, and `archived` are intentionally invalid. Repository deletion and transfer have no configuration representation. `has_downloads` is recognized in GitHub responses but is not a supported current update field.

## Custom Properties

`custom_properties` maps organization-defined property names to their repository values. Values may be strings, arrays of strings for multi-select properties, or `null` to explicitly unset a value. Quote values such as `"true"` and `"false"`; YAML booleans are rejected because GitHub custom-property values are strings.

The map is authoritative. A property returned by GitHub but omitted from a present map is unset. An empty map therefore requests that every currently set, removable custom property be unset. Organization and enterprise definitions, allowed values, required properties, and edit restrictions are not changed.

```yaml
custom_properties:
  status: active
  platforms:
    - linux
    - macos
  retired_at: null
```

Multi-select arrays are compared as selections rather than ordered lists, so a different response order does not create drift.

Property definitions stay under organization or enterprise control. **ghrepocfg** does not create definitions, change allowed values, relax edit restrictions, or alter whether a property is required. Names and allowed values must already exist in the destination organization or enterprise, which can limit portability between organizations.

Reading values requires repository read access. Writing requires repository administration or GitHub's repository-level **Custom properties: write** permission, and the property definition must allow the caller to edit its value. Restricted values can return `403 Forbidden`; values outside a select property's allowed set can return `422 Validation Failed`.

Each changed property is submitted independently. If one value is restricted or invalid, that path is reported as failed while unrelated property changes continue. A partially successful apply exits with an error and retains successful changes.

## Security

Boolean settings:

| Key | Description |
|---|---|
| `vulnerability_alerts` | Dependabot vulnerability alerts |
| `automated_security_fixes` | Dependabot security updates |

The following features use an object with `status: enabled` or `status: disabled`:

- `advanced_security`
- `code_security`
- `secret_scanning`
- `secret_scanning_push_protection`
- `secret_scanning_ai_detection`
- `secret_scanning_non_provider_patterns`
- `secret_scanning_delegated_alert_dismissal`
- `secret_scanning_delegated_bypass`

`secret_scanning_delegated_bypass_options.reviewers` is an array with:

| Key | Type or Values |
|---|---|
| `reviewer_id` | integer team or role ID |
| `reviewer_type` | `TEAM`, `ROLE` |
| `mode` | `ALWAYS`, `EXEMPT` |

Reviewer IDs are organization-specific and are an unavoidable portability exception. GitHub licensing and organization policy determine which fields are available.

## Actions

| Key | Type or Values |
|---|---|
| `enabled` | boolean |
| `allowed_actions` | `all`, `local_only`, `selected` |
| `selected_actions.github_owned_allowed` | boolean |
| `selected_actions.verified_allowed` | boolean |
| `selected_actions.patterns_allowed` | array of action patterns |
| `default_workflow_permissions` | `read`, `write` |
| `can_approve_pull_request_reviews` | boolean |

GitHub returns `409 Conflict` when selected-action details are read while selected actions are inactive. Full export therefore includes `selected_actions` only when the live policy is `selected`.

## Collaborators

`collaborators` is keyed by GitHub login. `permission` accepts `pull`, `triage`, `push`, `maintain`, `admin`, or `custom:ROLE NAME` for a custom repository role.

The collection includes active direct collaborators and pending invitations. It does not include access inherited from teams or organization base permissions.

## Teams

`teams` is keyed by organization team slug. `permission` accepts the same built-in or `custom:ROLE NAME` values as collaborators.

The present map is authoritative for repository team associations returned by GitHub. Parent-team or organization policy may prevent effective removal; GitHub reports such conflicts as mutation failures.

## Rulesets

Rulesets are keyed by name because numeric ruleset IDs are not portable. Names must be unique in the repository. Only rulesets whose source type is `Repository` are managed.

### Ruleset Fields

| Key | Type or Values |
|---|---|
| `target` | `branch` (default), `tag`, `push` |
| `enforcement` | `disabled`, `active`, `evaluate` |
| `bypass_actors` | array of bypass actor objects |
| `conditions.ref_name.include` | array of ref patterns |
| `conditions.ref_name.exclude` | array of ref patterns |
| `rules` | array of rule objects |

Ref patterns support GitHub values such as `~DEFAULT_BRANCH` and `~ALL`.

### Bypass Actors

| Key | Type or Values |
|---|---|
| `actor_id` | integer or null, depending on actor type |
| `actor_type` | `Integration`, `OrganizationAdmin`, `RepositoryRole`, `Team`, `DeployKey`, `User` |
| `bypass_mode` | `always` (default), `pull_request`, `exempt` |

Actor IDs are organization- or repository-specific portability exceptions.

### Rule Types and Parameters

- Parameterless: `creation`, `deletion`, `required_linear_history`, `required_signatures`, `non_fast_forward`, `license_compliance_scanning`
- `update`: `update_allows_fetch_and_merge`
- `merge_queue`: `check_response_timeout_minutes`, `grouping_strategy`, `max_entries_to_build`, `max_entries_to_merge`, `merge_method`, `min_entries_to_merge`, `min_entries_to_merge_wait_minutes`
- `required_deployments`: `required_deployment_environments`
- `pull_request`: `allowed_merge_methods`, `dismiss_stale_reviews_on_push`, `dismissal_restriction.allowed_actors`, `require_code_owner_review`, `require_last_push_approval`, `required_approving_review_count`, `required_review_thread_resolution`, `required_reviewers`
- `required_status_checks`: `required_status_checks`, `strict_required_status_checks_policy`, `do_not_enforce_on_create`
- Pattern rules `commit_message_pattern`, `commit_author_email_pattern`, `committer_email_pattern`, `branch_name_pattern`, `tag_name_pattern`: `name`, `negate`, `operator`, `pattern`
- `workflows`: `do_not_enforce_on_create`, `workflows`
- `code_scanning`: `code_scanning_tools`
- `copilot_code_review`: `review_draft_pull_requests`, `review_on_push`
- `file_path_restriction`: `restricted_file_paths`
- `max_file_path_length`: `max_file_path_length`
- `file_extension_restriction`: `restricted_file_extensions`
- `max_file_size`: `max_file_size`

Nested object shapes:

- `dismissal_restriction.allowed_actors`: objects with `id` and `type`
- `required_reviewers`: objects with `file_patterns`, `minimum_approvals`, and `reviewer` containing `id` and `type`
- `required_status_checks`: objects with `context` and optional `integration_id`
- `workflows`: objects with `path`, `repository_id`, optional `ref`, and optional `sha`
- `code_scanning_tools`: objects with `tool`, `security_alerts_threshold`, and `alerts_threshold`

Workflow repository IDs, reviewer IDs, integration IDs, and actor IDs may limit portability. GitHub may omit a rule's parameters when every parameter uses its default; this is valid YAML and is preserved by export. For `update`, an omitted `update_allows_fetch_and_merge` value and explicit `false` compare as the same GitHub default, while explicit `true` remains a managed requirement. Unsupported rule types, misspelled keys, and parameters used with the wrong rule type fail validation.

## Generated YAML

Generated files use stable section ordering and normalized formatting. Existing comments and hand-crafted formatting are not preserved. Semantic management scope is preserved unless `export --full` is used.
