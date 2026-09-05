---
layout: default
title: Custom Properties
nav_order: 7
permalink: /custom-properties
---

# Custom Properties
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Repository Metadata as Code

**ghrepocfg** exports and applies the custom-property values assigned to a repository. This makes ownership, lifecycle, service tier, platform, and other organization-defined metadata reviewable alongside the repository settings they describe.

Property definitions stay under organization or enterprise control. **ghrepocfg** does not create definitions, change allowed values, relax edit restrictions, or alter whether a property is required.

## Configuration

Use the top-level `custom_properties` map. GitHub supports string values, arrays of strings for multi-select properties, and `null` to unset a value.

```yaml
custom_properties:
  status: active
  service_tier: tier-1
  platforms: [linux, macos]
  deprecated_on: null
```

Quote boolean-looking strings:

```yaml
custom_properties:
  customer_facing: "true"
```

Unquoted YAML booleans and numeric or object values fail local validation before GitHub is contacted.

## Authoritative Behavior

A present `custom_properties` map is authoritative:

- changed or new entries are set;
- entries present on GitHub but absent from the map are unset with a `null` API value;
- an explicit `null` also requests an unset;
- an omitted `custom_properties` section leaves every property unmanaged.

Multi-select arrays are compared as selections rather than ordered lists, so a different response order does not create drift.

An empty map attempts to unset all currently assigned, removable values. Required properties or properties whose values are controlled by organization actors may reject that request.

## Export

A full export includes all custom-property values returned for the repository. A scope-preserving export refreshes them only when the existing file already contains `custom_properties`.

```console
ghrepocfg export --repo OWNER/REPO --full
ghrepocfg export --repo OWNER/REPO --config .ghrepocfg.yaml
```

Reading repository custom-property values requires repository read access. Public repositories may expose values without authentication, subject to GitHub policy.

## Apply and Permissions

Writing requires repository administration or GitHub's repository-level **Custom properties: write** permission. The property definition must also permit repository actors to edit its value. Organization- or enterprise-restricted values can return `403 Forbidden`; values outside a select property's allowed set can return `422 Validation Failed`.

**ghrepocfg** plans and submits each property independently. If one value is restricted or invalid, its path is reported as failed and application continues with the remaining planned changes. A partially successful apply exits with an error; run a dry-run or apply again to see the remaining drift.

```console
ghrepocfg apply --dry-run
ghrepocfg apply --yes
```

## Portability

Property names and allowed values must exist in the destination repository's organization or enterprise. A configuration exported from one organization may therefore require matching definitions before it can be applied elsewhere. Definitions, defaults, required flags, and editing policies are intentionally outside repository configuration.
