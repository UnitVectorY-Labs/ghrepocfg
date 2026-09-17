---
layout: default
title: Permissions
nav_order: 9
permalink: /permissions
---

# Permissions and Partial Access

Access is evaluated per endpoint. A token's permissions, its selected repositories, the account's role, organization policy, and feature availability all matter. Repository admin status alone does not establish token permissions.

## Default Behavior

- The initial repository metadata read must succeed; otherwise the command fails.
- A denied read (`403`, excluding rate limits) or ambiguous absence (`404`) marks affected state unavailable.
- A new/full export omits unavailable paths. A scoped refresh preserves their existing values and warns that they were not refreshed.
- Apply skips unreadable paths and denied writes while continuing independent operations. Dependent writes are skipped when their prerequisite fails.
- Unknown values never become empty collections, false toggles, or disabled features. Incomplete pagination invalidates the collection. Collaborators require both active access and invitation reads. Hidden ruleset bypass actors invalidate authoritative ruleset state.
- Authentication, rate limits, transport/server/decode errors, conflicts (`409`), and validation failures (`422`) remain errors. A `404` during mutation remains an error.
- A failed multi-step operation that already made changes is a partial failure, even if the last response was a permission denial.

Warnings go to stderr and identify affected attributes. JSON dry runs include `complete` and `skipped` (`path` and `reason`). Default exit `0` can include skips; exit `2` means known dry-run drift. Incomplete evaluation is never reported as an unqualified “No changes.”

## Strict Mode

`--strict` defaults to false. Strict export requires complete requested state before emitting YAML or replacing a file. Strict apply fails before mutation on incomplete reads, stops on the first failed/denied mutation, and verifies all configured state after writing. Verification drift, unreadable values, and pending invitations fail with exit `1`. Code scanning setup is polled for up to 15 seconds within the overall command timeout; other mismatches fail immediately.

Strictness applies to the supplied configuration's management scope, including authoritative collections. Full export requests all supported applicable domains. Settings known not to apply, such as private-only Actions settings on public repositories, are excluded from full discovery. A strict dry run cannot prove future write permission. GitHub provides no transaction or rollback across these APIs.

## Fine-Grained PAT and GitHub App Permissions

Paths below are relative to `/repos/{owner}/{repo}`, except the organization-team write endpoint. Read/write permissions are token requirements; endpoint-specific account roles and visibility restrictions still apply.

| Attributes | API | Read | Write |
|---|---|---|---|
| Repository settings and topics | GET/PATCH repository; PUT `/topics` | Metadata | Administration |
| Custom properties | GET/PATCH `/properties/values` | Metadata | Custom properties |
| Security analysis | GET/PATCH repository | Metadata plus admin/security-manager visibility | Administration plus authorized role |
| Vulnerability alerts, security fixes, immutable releases | GET/PUT/DELETE `/vulnerability-alerts`, `/automated-security-fixes`, `/immutable-releases` | Administration | Administration |
| Private vulnerability reporting | GET/PUT/DELETE `/private-vulnerability-reporting` | Metadata | Administration |
| Code scanning default setup | GET/PATCH `/code-scanning/default-setup` | Administration | Administration |
| Actions policies, selected actions, workflow defaults, retention, fork policies, sharing | GET/PUT `/actions/permissions` and its setting endpoints | Administration | Administration |
| OIDC subject | GET/PUT `/actions/oidc/customization/sub` | Actions | Actions |
| Cache retention | GET/PUT `/actions/cache/retention-limit` | Administration | Administration |
| Cache storage | GET/PUT `/actions/cache/storage-limit` | Actions | Administration |
| Repository variables | GET/POST/PATCH/DELETE `/actions/variables` and named variables | Variables | Variables |
| Collaborators | GET/PUT/DELETE `/collaborators` and named collaborators | Metadata, with account-role restrictions | Administration |
| Invitations | GET `/invitations`, DELETE named invitation | Administration, or documented Private repository invitations user permission | Administration |
| Teams | GET `/teams`; PUT/DELETE `/orgs/{org}/teams/{slug}/repos/{owner}/{repo}` | Administration | Administration and organization Members read, plus Metadata read |
| Rulesets | GET/POST/PUT/DELETE `/rulesets` and individual rulesets | Metadata; bypass visibility requires ruleset write access | Administration |
| Environments and branch/tag policies | GET/PUT/DELETE `/environments`; policy GET/POST/PUT/DELETE | Actions | Administration |
| Environment variables | GET/POST/PATCH/DELETE `/environments/{name}/variables` and named variables | Environments | Environments |
| Pages | GET/POST/PUT/DELETE `/pages` | Pages | Pages and Administration |
| Labels | GET/POST/PATCH/DELETE `/labels` and named labels | Issues or Pull requests | Issues or Pull requests |
| Autolinks and deploy keys | GET/POST/DELETE `/autolinks`, `/keys` and individual entries | Administration | Administration |

“Read” means the named permission at read level; “Write” means write level unless explicitly noted otherwise. Successful repository reads can still omit individual security/settings fields. Such omissions do not authorize guessing their values.

References: [repositories](https://docs.github.com/en/rest/repos/repos), [custom properties](https://docs.github.com/en/rest/repos/custom-properties), [Actions policies](https://docs.github.com/en/rest/actions/permissions), [OIDC](https://docs.github.com/en/rest/actions/oidc), [cache](https://docs.github.com/en/rest/actions/cache), [variables](https://docs.github.com/en/rest/actions/variables), [collaborators](https://docs.github.com/en/rest/collaborators/collaborators), [invitations](https://docs.github.com/en/rest/collaborators/invitations), [teams](https://docs.github.com/en/rest/teams/teams), [rulesets](https://docs.github.com/en/rest/repos/rules), [environments](https://docs.github.com/en/rest/deployments/environments), [branch policies](https://docs.github.com/en/rest/deployments/branch-policies), [Pages](https://docs.github.com/en/rest/pages/pages), [labels](https://docs.github.com/en/rest/issues/labels), [autolinks](https://docs.github.com/en/rest/repos/autolinks), [deploy keys](https://docs.github.com/en/rest/deploy-keys/deploy-keys), [code scanning](https://docs.github.com/en/rest/code-scanning/code-scanning).

## Classic PATs and OAuth Tokens

Classic scopes are coarser. `repo` generally covers private repository management; some public operations permit `public_repo`, while settings endpoints may explicitly require `repo`. Collaborator listing documents `repo` and `read:org`. Team visibility also depends on organization access. `workflow` is not a substitute for repository settings permissions.

GitHub's cache-limit documentation currently names `admin:repository`, absent from the published OAuth scope list; do not assume it is an available scope. Prefer fine-grained permissions and actual response diagnostics. See [OAuth scopes](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/scopes-for-oauth-apps).

## Diagnosing Access

Warnings use `X-Accepted-GitHub-Permissions` when available, otherwise an endpoint-specific diagnostic hint. That header describes required permissions, not granted permissions. The client retains response headers for scope/rate-limit diagnostics. A readable repository does not prove that every child resource is readable.

GitHub can use `404` to conceal access denial. Pages and vulnerability-alert status therefore remain unknown on an ambiguous `404`; they are not exported as disabled. A `403` rate limit must be retried after the limit resets, not ignored as an inaccessible attribute. See [GitHub REST troubleshooting](https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api).
