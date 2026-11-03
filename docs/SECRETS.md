---
layout: default
title: Encrypted Secrets
nav_order: 10
permalink: /secrets
---

# Encrypted Secrets

Commit locally encrypted repository secrets to YAML, review the deployment plan, and upload them with a separate command:

```bash
ghrepocfg secrets apply --repo OWNER/REPO --config secrets.yaml --dry-run
ghrepocfg secrets apply --repo OWNER/REPO --config secrets.yaml --yes
```

Repository secrets for **Actions, Agents, Codespaces, and Dependabot** are supported. Environment, organization, and user secrets are outside this command's scope. The command processes all configured application sections in one invocation. Other settings in the same YAML file are validated but are applied only by ordinary `ghrepocfg apply`.

## Prepare the YAML with GitHub CLI

GitHub CLI encrypts locally with the target repository's public key. `--no-store` prints Base64 ciphertext without uploading it. It still needs authenticated network access to fetch that key. Supply plaintext through the interactive prompt or stdin; do not put credentials in command-line arguments.

This shell example captures the key ID separately because `gh secret set --no-store` does not output it. It compares the public key before and after encryption to detect a key change between requests:

```bash
set -e
secret_target=OWNER/REPO
secret_application=actions # actions, agents, codespaces, or dependabot
secret_key_before=$(gh api "repos/$secret_target/$secret_application/secrets/public-key" \
  --jq '[.key_id, .key] | join(":")')
secret_ciphertext=$(gh secret set DEMO_SECRET --repo "$secret_target" \
  --app "$secret_application" --no-store)
secret_key_after=$(gh api "repos/$secret_target/$secret_application/secrets/public-key" \
  --jq '[.key_id, .key] | join(":")')
if [ "$secret_key_before" != "$secret_key_after" ]; then
  echo 'Public key changed; repeat encryption.' >&2
  exit 1
fi
secret_key_id=${secret_key_before%%:*}
cat > secrets.yaml <<YAML
$secret_application:
  secrets:
    DEMO_SECRET:
      repository: $secret_target
      key_id: "$secret_key_id"
      encrypted_value: "$secret_ciphertext"
YAML
unset secret_ciphertext
```

The resulting file contains ciphertext and public metadata, and can be committed. Retain the original credential in a recoverable source such as a password manager: ghrepocfg cannot decrypt the artifact or re-encrypt it for another key.

## Application Binding

The containing section selects the application and its API namespace. The same secret name can have independent values in all four sections. Each repository/application pair needs its own encrypted artifact, prepared using the matching `gh secret set --app` value. There is no implicit fallback or copying between applications, and `secrets apply` has no application override flag.

| YAML map | `gh --app` value | Repository API prefix | Fine-grained permissions: key read / upload |
|---|---|---|---|
| `actions.secrets` | `actions` | `/repos/OWNER/REPO/actions/secrets` | Secrets: read / write |
| `agents.secrets` | `agents` | `/repos/OWNER/REPO/agents/secrets` | Agent secrets: read / write |
| `codespaces.secrets` | `codespaces` | `/repos/OWNER/REPO/codespaces/secrets` | Codespaces secrets: write / write |
| `dependabot.secrets` | `dependabot` | `/repos/OWNER/REPO/dependabot/secrets` | Dependabot secrets: read / write |

Codespaces requires write permission even to retrieve its repository public key. Account-role requirements also apply; repository Codespaces secret writes require repository admin access.

## Configuration

```yaml
actions:
  secrets:
    DEMO_SECRET:
      repository: OWNER/REPO
      key_id: "KEY_ID_USED_FOR_ENCRYPTION"
      encrypted_value: "BASE64_SEALED_BOX"
```

Use the same entry shape under `agents.secrets`, `codespaces.secrets`, and `dependabot.secrets`, with ciphertext and key ID generated for that application. All three entry fields are required. `repository` must be a literal `OWNER/REPO` matching the invocation's target, without case sensitivity. Every target repository/application pair needs its own encrypted artifact. Names follow GitHub secret naming rules, and names differing only by case are rejected. Ciphertext must be standard Base64 on one line and decode to a sealed box of 48 through 49,200 bytes (48 bytes overhead plus at most 48 KiB plaintext). Structural validation cannot prove that a ciphertext was encrypted for the claimed key; GitHub processes the submitted payload.

**Secrets are an exception to authoritative collection management.** Entries request uploads only. Omitted names and `secrets: {}` never delete existing secrets. This version provides no secret deletion operation.

## Deployment and Comparison

Each invocation uploads every configured entry, even when its ciphertext is unchanged. There is no deployment ledger, companion variable, plaintext fingerprint, or comparison against Git history.

GitHub returns secret names and timestamps, but neither stored plaintext nor ciphertext nor a value fingerprint. Consequently, the plan says **upload configured encrypted value** and does not claim detected value drift. Encryption is randomized: regenerating an artifact from identical plaintext can produce different ciphertext.

The command validates all target bindings and key IDs before confirmation, then checks the current public key again before each upload. A mismatch requires re-encryption; changing only `key_id` cannot repair ciphertext. A key change or other failure can leave earlier uploads applied. There is no transaction or rollback.

Ordinary `apply` skips nonempty application secret maps with a warning directing you to `secrets apply`, while continuing independent settings. Its `--strict` mode rejects this incomplete scope before mutations.

## Flags, Results, and Permissions

`secrets apply` supports the usual `--repo`/`-R`, `--config`, `--dry-run`, `--json`, `--yes`/`-y`, `--verbose`/`-v`, `--strict`, and help flags. Repository, configuration-path, and authentication resolution follow [Usage](USAGE.md). No CLI runtime dependency is needed to upload ciphertext; `gh` is used here to prepare it.

- Dry runs never prompt or upload. Exit `2` means uploads are planned, **not** that different secret values were detected.
- JSON is available only with `--dry-run`. It includes `repository`, `preflight_complete`, `value_comparison: unavailable`, `uploads` containing `{path, operation: upload}`, and optional `skipped` entries. Preflight completeness does not establish value equality or future write access.
- Plans and success messages omit ciphertext. A successful upload is reported as accepted by GitHub, with the value unverified.
- Public-key reads and uploads require the application-specific permissions in the table above, plus applicable account access. Fine-grained PATs and GitHub App tokens can provide these permissions; classic PATs require the applicable repository scope.
- Permission-denied reads skip only the affected application; other application keys and uploads are evaluated independently. Permission-denied writes warn and continue independent uploads. Exit `0` may include explicit skips. Rate limits, malformed responses, key mismatches, validation errors, and other API failures return exit `1`; earlier successful uploads remain applied.
- `--strict` is off by default. With configured secret entries, it returns exit `1` without uploading, including for dry runs: GitHub cannot provide the value verification required by strict apply. An omitted or empty secrets map requires no verification and succeeds without API requests.

## Export

GitHub cannot export encrypted secret values. A new export warns that secret values are excluded from discovery and does not discover any application secret map. Existing encrypted entries are retained during both scoped export and `export --full`, with an explicit warning that they were not refreshed. They are never replaced with an empty collection. An existing destination must therefore be valid YAML even for `--full`.

Strict export of a file containing nonempty secret entries fails before replacing the file because requested values cannot be refreshed. Empty secret maps are retained without a warning. Full discovery without configured secret entries covers readable settings; secret values remain outside discovery.

References: [GitHub CLI secret set](https://cli.github.com/manual/gh_secret_set), [Actions secrets API](https://docs.github.com/en/rest/actions/secrets), [Agents secrets API](https://docs.github.com/en/rest/agents/secrets), [Codespaces repository secrets API](https://docs.github.com/en/rest/codespaces/repository-secrets), [Dependabot secrets API](https://docs.github.com/en/rest/dependabot/secrets), [secret encryption](https://docs.github.com/en/rest/guides/encrypting-secrets-for-the-rest-api).
