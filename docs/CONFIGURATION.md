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
| `environments` | map keyed by environment name | Authoritative environment membership; omitted fields within a retained environment are unmanaged |
| `pages` | object | Site settings; `enabled: false` removes the site |
| `labels` | map keyed by label name | Authoritative label definitions |
| `autolinks` | map keyed by prefix | Authoritative autolink definitions |
| `deploy_keys` | map keyed by unique title | Authoritative public deploy keys |
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
| `immutable_releases` | boolean | Enable release immutability through its dedicated API; owner policy can prevent changes |
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
| `automated_security_fixes` | Dependabot security updates; reads the API's `enabled` value, not its HTTP success status |
| `private_vulnerability_reporting` | Allow private vulnerability submissions |

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

### Code Scanning Default Setup

`security.code_scanning_default_setup` configures GitHub's default CodeQL setup:

| Key | Type or Values |
|---|---|
| `state` | `configured`, `not-configured` |
| `languages` | Array of GitHub CodeQL language identifiers, such as `go` |
| `query_suite` | `default`, `extended` |
| `threat_model` | `remote`, `remote_and_local` |
| `runner_type` | `standard`, `labeled` |
| `runner_label` | String identifying a self-hosted runner label |

Omitted fields remain unmanaged. Setup changes can be asynchronous; rerun dry-run after GitHub finishes configuring the scanner. Code Security must already be available before explicitly managing this section. Disabling setup requires only `state: not-configured`, without other scanner settings. The API's read-only schedule and update timestamp are not YAML settings.

## Actions

| Key | Type or Values |
|---|---|
| `sha_pinning_required` | boolean |
| `artifact_and_log_retention_days` | positive integer |
| `fork_pr_contributor_approval` | `first_time_contributors_new_to_github`, `first_time_contributors`, `all_external_contributors` |
| `access_level` | `none`, `user`, `organization`, `enterprise`, subject to GitHub's repository/owner restrictions |
| `private_fork_workflows.run_workflows_from_fork_pull_requests` | boolean |
| `private_fork_workflows.send_write_tokens_to_workflows` | boolean |
| `private_fork_workflows.send_secrets_and_variables` | boolean |
| `private_fork_workflows.require_approval_for_fork_pr_workflows` | boolean |
| `oidc.use_default` | boolean |
| `oidc.include_claim_keys` | ordered array of OIDC subject claim keys; use with `use_default: false` |
| `oidc.use_immutable_subject` | boolean |
| `cache.max_retention_days` | positive integer |
| `cache.max_size_gb` | positive integer |
| `variables` | authoritative map of variable names to string values |
| `enabled` | boolean |
| `allowed_actions` | `all`, `local_only`, `selected` |
| `selected_actions.github_owned_allowed` | boolean |
| `selected_actions.verified_allowed` | boolean |
| `selected_actions.patterns_allowed` | array of action patterns |
| `default_workflow_permissions` | `read`, `write` |
| `can_approve_pull_request_reviews` | boolean |

GitHub returns `409 Conflict` when selected-action details are read while selected actions are inactive. Full export therefore includes `selected_actions` only when the live policy is `selected`.

## Environments

`environments` is keyed by environment name. Environment names are compared without case sensitivity. An empty map deletes all environments returned by GitHub. Deleting an environment also deletes its associated configuration, including secrets that ghrepocfg cannot export. Within a retained environment, omitted fields and variables remain unchanged.

| Key | Type or Values |
|---|---|
| `wait_timer` | Minutes, 0 through 43200; 0 removes the delay |
| `prevent_self_review` | boolean |
| `reviewers` | Up to six `{type: User or Team, id: integer}` entries; `[]` clears reviewers |
| `deployment_branch_policy.protected_branches` | boolean |
| `deployment_branch_policy.custom_branch_policies` | boolean |
| `deployment_branch_patterns` | Authoritative array of allowed branch patterns |
| `deployment_tag_patterns` | Authoritative array of allowed tag patterns |
| `variables` | Authoritative map of variable names to string values |

The policy flags cannot both be true. Both false means unrestricted deployment branches. Patterns require custom branch policies; creating an environment with patterns and no explicit policy selects custom policies. Switching away from custom policies stops managing their patterns. Reviewers refer to existing numeric user/team IDs. Reviewers and wait timers depend on GitHub plan and visibility.

Environment updates preserve omitted protection settings. Creation precedes policy and variable requests. One environment is one planned operation; a child request failure can leave earlier steps applied. Custom deployment protection integrations are outside this schema and are not removed from retained environments.

## Variables

Actions and environment variables contain readable string values. Quote numeric and boolean-looking values. Names use letters, digits, and underscores, cannot start with a digit or `GITHUB_`, and are compared without case sensitivity. Empty maps delete the respective variable collection. Secrets and organization-level variables are not managed.

## Pages

| Key | Type or Values |
|---|---|
| `enabled` | boolean; `false` deletes the Pages site |
| `build_type` | `workflow` or `legacy` |
| `source.branch` | Existing source branch for legacy publishing |
| `source.path` | `/` or `/docs` |
| `cname` | Custom domain; `""` removes it |
| `https_enforced` | boolean |

Site settings imply that a site should exist. `enabled: true` without publishing settings creates a workflow-based site. `enabled: false` must appear without other Pages settings. Workflow definitions, content, DNS records, and certificate provisioning are not managed. A missing site exports as `enabled: false`; administration visibility is required. HTTPS/domain changes may need GitHub's asynchronous provisioning to complete.

## Labels, Autolinks, and Deploy Keys

| Collection | Map key | Entry fields |
|---|---|---|
| `labels` | Label name | `color`: six hexadecimal digits without `#`; `description`: string |
| `autolinks` | Key prefix, e.g. `ENG-` | `url_template`: URL containing `<num>`; `is_alphanumeric`: boolean |
| `deploy_keys` | Unique title | `key`: SSH public key; `read_only`: boolean |

These maps are authoritative, including `{}`. Set `read_only: true` explicitly for read-only deploy keys; false grants write access. Private key material is never read or generated by ghrepocfg. SSH public-key comments do not cause drift. Label names and colors compare without case sensitivity. Duplicate label or environment names differing only in case are rejected.

Autolinks and deploy keys have no update API. Plans mark changed entries as `replace` (delete then create); if deletion fails, creation is not attempted. If creation fails after deletion, rerun after correcting the error. Duplicate live deploy-key titles or autolink prefixes prevent authoritative export and apply because YAML cannot identify them uniquely.

## Collaborators

`collaborators` is keyed by GitHub login. `permission` accepts `pull`, `triage`, `push`, `maintain`, `admin`, or `custom:ROLE NAME` for a custom repository role.

The collection includes active direct collaborators and pending invitations. It does not include access inherited from teams or organization base permissions.

## Teams

`teams` is keyed by organization team slug. `permission` accepts the same built-in or `custom:ROLE NAME` values as collaborators.

The present map is authoritative for direct organization-team repository associations. Teams marked with organization/enterprise access sources and enterprise-owned teams are excluded. Older responses without provenance retain the existing organization-team behavior. Parent-team or organization policy may prevent effective removal; GitHub reports such conflicts as mutation failures.

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
