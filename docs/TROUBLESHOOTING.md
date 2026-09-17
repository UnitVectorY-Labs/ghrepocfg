---
layout: default
title: Troubleshooting
nav_order: 8
permalink: /troubleshooting
---

# Troubleshooting
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Repository Cannot Be Determined

Specify the target explicitly:

```bash
ghrepocfg apply --repo OWNER/REPO
```

Inference requires a current Git checkout with exactly one distinct GitHub.com repository among its configured remotes. `GHREPOCFG_REPO` is another fallback.

## Configuration File Cannot Be Found

Use `--config PATH` or `GHREPOCFG_CONFIG`. The default is `.ghrepocfg.yaml` at the Git root, or in the working directory for apply outside a checkout.

## Authentication Fails

Check the active GitHub CLI identity first because it has credential precedence:

```bash
gh auth status
```

If the GitHub CLI is unavailable, set `GH_TOKEN` or `GITHUB_TOKEN`. See [Installation](INSTALL.md#authentication).

## GitHub Returns 403 or 404

Verify repository selection, account role, organization policy, and the endpoint-specific token permissions in [Permissions](PERMISSIONS.md). There is no blanket admin requirement for reads.

Custom-property reads require Metadata read. Fine-grained tokens need Custom properties write to update values, and the account role/property definition must also permit editing.

GitHub may use `404` to hide an inaccessible resource. **ghrepocfg** marks that state unavailable instead of treating it as empty or disabled. Default operation warns and continues; `--strict` fails. Rate-limit `403` responses remain operational errors.

## GitHub Returns 409 for Selected Actions

GitHub does not expose selected-action details while `allowed_actions` is `all` or `local_only`. First apply a configuration that sets:

```yaml
actions:
  allowed_actions: selected
```

Then add `selected_actions` policy in a subsequent configuration.

## GitHub Rejects a Supported Setting

Repository visibility, organization policy, product licensing, fork status, or another GitHub constraint may make a supported repository-level field unavailable for a particular repository. The error is reported for that path, and unrelated independent mutations continue.

For custom properties, confirm that the property exists, the value is allowed, the property is not restricted to other organization or enterprise actors, and required properties are not being unset. A rejected custom property does not prevent other planned property requests from running.

## Dry-Run Returns a Nonzero Status

- Exit `2` means drift or export-file changes were successfully detected.
- Exit `1` means an operational failure or incomplete strict evaluation.
- Default exit `0` can include permission skips; inspect warnings or JSON `complete` and `skipped`.

Inspect JSON without treating expected drift as an operational failure by handling exit `2` separately. See [Examples](EXAMPLES.md#use-json-in-ci).

## Apply Was Partially Successful

Review the consolidated failure list. Successful independent operations are not rolled back. Correct the failure, run dry-run again to see the remaining drift, and reapply.

## Legacy Protection Warning

Legacy branch or tag protection is present but unmanaged. Review it directly in GitHub. **ghrepocfg** never converts legacy protection into rulesets automatically.

## Unknown YAML Key

Unknown keys are intentionally fatal. Check spelling and compare the key with [Configuration Reference](CONFIGURATION.md). Visibility, archive, identity, and secrets keys produce specific errors because they are intentionally out of scope.

## Additional Settings Are Omitted from Full Export

Read the warning accompanying the omitted group. Public-fork approval policy is unavailable for private repositories; private vulnerability reporting, CodeQL setup, and deployment protections depend on visibility, licensing, and owner policy. Full discovery omits inaccessible groups; a scoped refresh retains previous values with a warning. Strict mode rejects either incomplete result. Remove a section only when you intend to stop managing it.

An owner may forbid a private-fork workflow policy change with `422` even though its GET endpoint is readable. The application reports the failure and continues independent changes. It never changes organization policy to bypass the restriction.

## A Replacement or Environment Change Partially Failed

Autolinks and deploy keys require delete-then-create replacement. Environment protection operations precede dependent creation requests. Variable-only updates use the variable API without rewriting environment protection settings. A later failure does not roll back earlier API calls. Correct the reported error and rerun dry-run; live state determines the remaining work.

## Cache Limit Write Is Accepted but Does Not Take Effect

GitHub can return success for a cache-limit update while the GET endpoint still returns another effective value. ghrepocfg reads cache limits back after writing and reports a failure if they do not match. Recheck the repository and owner policy and rerun dry-run; do not assume HTTP success means the requested limit took effect. During live validation, requesting three retention days returned success but continued reporting seven; the API responses did not establish the underlying cause.

## Private Actions Policies on a Public Repository

Full export automatically omits `actions.private_fork_workflows` and `actions.access_level` for public repositories without warnings. If a configuration copied from a private repository includes those fields, remove them before applying or refreshing that scoped configuration against a public repository. These policies do not apply there; the application reports this before calling their endpoints.

## Strict Verification Fails

A successful HTTP write does not prove that GitHub made the requested value effective. Strict apply reads the managed state again and fails on drift or incomplete verification. Code scanning setup receives bounded polling; other mismatches fail immediately. Pending collaborator invitations are reported as pending rather than verified access. Earlier mutations remain applied: inspect the named paths and rerun dry-run after correcting access or policy.

A `404` from Pages or vulnerability-alert status is ambiguous even when the repository is readable. The attribute is skipped instead of exported as disabled. Strict mode cannot certify that attribute until its state can be established.
