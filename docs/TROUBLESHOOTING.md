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

Verify that the token can see the repository and has the permissions required by every managed section. Full export and authoritative security, access, and ruleset sections require repository admin visibility.

Custom-property writes additionally require repository administration or the repository-level **Custom properties: write** permission, and the organization or enterprise definition must allow the caller to edit values.

GitHub may use `404` to hide an inaccessible resource. **ghrepocfg** fails rather than treating ambiguous authoritative state as empty.

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
- Exit `1` means validation, authentication, permission, API, or another operational failure.

Inspect JSON without treating expected drift as an operational failure by handling exit `2` separately. See [Examples](EXAMPLES.md#use-json-in-ci).

## Apply Was Partially Successful

Review the consolidated failure list. Successful independent operations are not rolled back. Correct the failure, run dry-run again to see the remaining drift, and reapply.

## Legacy Protection Warning

Legacy branch or tag protection is present but unmanaged. Review it directly in GitHub. **ghrepocfg** never converts legacy protection into rulesets automatically.

## Unknown YAML Key

Unknown keys are intentionally fatal. Check spelling and compare the key with [Configuration Reference](CONFIGURATION.md). Visibility, archive, identity, and secrets keys produce specific errors because they are intentionally out of scope.
