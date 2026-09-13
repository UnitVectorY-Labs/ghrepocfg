package reconcile

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

func additionalChanges(p *Plan, owner, repo string, d *config.Config, s *github.State, e Executor) {
	change := func(path, method, suffix string, before, after, body any) {
		if valuesEqual(before, after) {
			return
		}
		add(p, Change{Modify, path, before, after, func(ctx context.Context) error { return e.Mutate(ctx, owner, repo, method, suffix, body) }})
	}
	toggle := func(path, suffix string, want, got *bool) {
		if want == nil {
			return
		}
		method := http.MethodDelete
		if *want {
			method = http.MethodPut
		}
		change(path, method, suffix, indirect(got), *want, nil)
	}
	if d.Repository != nil {
		toggle("repository.immutable_releases", "/immutable-releases", d.Repository.ImmutableReleases, s.Repository.ImmutableReleases)
	}
	if d.Security != nil {
		toggle("security.private_vulnerability_reporting", "/private-vulnerability-reporting", d.Security.PrivateVulnerabilityReporting, s.Security.PrivateVulnerabilityReporting)
		if want := d.Security.CodeScanningDefaultSetup; want != nil {
			body := object(want)
			if want.State != nil && *want.State == "not-configured" {
				body = map[string]any{"state": "not-configured"}
			}
			change("security.code_scanning_default_setup", http.MethodPatch, "/code-scanning/default-setup", config.Project(want, s.Security.CodeScanningDefaultSetup), want, body)
		}
	}
	if a := d.Actions; a != nil {
		got := s.Actions
		if a.ArtifactAndLogRetentionDays != nil {
			change("actions.artifact_and_log_retention_days", http.MethodPut, "/actions/permissions/artifact-and-log-retention", indirect(got.ArtifactAndLogRetentionDays), *a.ArtifactAndLogRetentionDays, map[string]any{"days": *a.ArtifactAndLogRetentionDays})
		}
		if a.ForkPRContributorApproval != nil {
			change("actions.fork_pr_contributor_approval", http.MethodPut, "/actions/permissions/fork-pr-contributor-approval", indirect(got.ForkPRContributorApproval), *a.ForkPRContributorApproval, map[string]any{"approval_policy": *a.ForkPRContributorApproval})
		}
		if a.AccessLevel != nil {
			change("actions.access_level", http.MethodPut, "/actions/permissions/access", indirect(got.AccessLevel), *a.AccessLevel, map[string]any{"access_level": *a.AccessLevel})
		}
		if a.PrivateForkWorkflows != nil {
			change("actions.private_fork_workflows", http.MethodPut, "/actions/permissions/fork-pr-workflows-private-repos", config.Project(a.PrivateForkWorkflows, got.PrivateForkWorkflows), a.PrivateForkWorkflows, merged(got.PrivateForkWorkflows, a.PrivateForkWorkflows))
		}
		if a.OIDC != nil {
			body := merged(got.OIDC, a.OIDC)
			if use, ok := body["use_default"].(bool); ok && use {
				delete(body, "include_claim_keys")
			}
			change("actions.oidc", http.MethodPut, "/actions/oidc/customization/sub", config.Project(a.OIDC, got.OIDC), a.OIDC, body)
		}
		if a.Cache != nil {
			cur := got.Cache
			if cur == nil {
				cur = &config.CacheSettings{}
			}
			if a.Cache.MaxRetentionDays != nil {
				change("actions.cache.max_retention_days", http.MethodPut, "/actions/cache/retention-limit", indirect(cur.MaxRetentionDays), *a.Cache.MaxRetentionDays, map[string]any{"max_cache_retention_days": *a.Cache.MaxRetentionDays})
			}
			if a.Cache.MaxSizeGB != nil {
				change("actions.cache.max_size_gb", http.MethodPut, "/actions/cache/storage-limit", indirect(cur.MaxSizeGB), *a.Cache.MaxSizeGB, map[string]any{"max_cache_size_gb": *a.Cache.MaxSizeGB})
			}
		}
		variableChanges(p, owner, repo, "actions.variables", "/actions/variables", a.Variables, got.Variables, e)
	}
	if d.Pages != nil {
		pagesChanges(p, owner, repo, d.Pages, s.Additional.Pages, e)
	}
	if d.Labels != nil {
		got := matchNames(*d.Labels, mapValue(s.Additional.Labels))
		for _, name := range unionKeys(*d.Labels, got) {
			want, wok := (*d.Labels)[name]
			cur, cok := got[name]
			want.Color = strings.ToLower(want.Color)
			cur.Color = strings.ToLower(cur.Color)
			if wok && cok && valuesEqual(want, cur) {
				continue
			}
			method, path, op := http.MethodPost, "/labels", Add
			var body any = map[string]any{"name": name, "color": want.Color, "description": want.Description}
			if cok {
				path += "/" + url.PathEscape(name)
				method, op = http.MethodPatch, Modify
			}
			if !wok {
				method, op, body = http.MethodDelete, Remove, nil
			}
			add(p, Change{op, "labels." + name, optional(cok, cur), optional(wok, want), func(ctx context.Context) error { return e.Mutate(ctx, owner, repo, method, path, body) }})
		}
	}
	if d.Autolinks != nil {
		got := mapValue(s.Additional.Autolinks)
		for _, name := range unionKeys(*d.Autolinks, got) {
			want, wok := (*d.Autolinks)[name]
			cur, cok := got[name]
			if wok && cok && valuesEqual(want, cur) {
				continue
			}
			id := s.AutolinkIDs[name]
			op := Add
			if cok {
				op = Replace
			}
			if !wok {
				op = Remove
			}
			add(p, Change{op, "autolinks." + name, optional(cok, cur), optional(wok, want), func(ctx context.Context) error {
				if cok {
					if err := e.Mutate(ctx, owner, repo, http.MethodDelete, "/autolinks/"+intString(id), nil); err != nil {
						return err
					}
				}
				if wok {
					return e.Mutate(ctx, owner, repo, http.MethodPost, "/autolinks", map[string]any{"key_prefix": name, "url_template": want.URLTemplate, "is_alphanumeric": want.IsAlphanumeric})
				}
				return nil
			}})
		}
	}
	if d.DeployKeys != nil {
		got := mapValue(s.Additional.DeployKeys)
		for _, name := range unionKeys(*d.DeployKeys, got) {
			want, wok := (*d.DeployKeys)[name]
			cur, cok := got[name]
			if wok && cok && keyMaterial(want.Key) == keyMaterial(cur.Key) && want.ReadOnly == cur.ReadOnly {
				continue
			}
			id := s.DeployKeyIDs[name]
			op := Add
			if cok {
				op = Replace
			}
			if !wok {
				op = Remove
			}
			add(p, Change{op, "deploy_keys." + name, optional(cok, cur), optional(wok, want), func(ctx context.Context) error {
				if cok {
					if err := e.Mutate(ctx, owner, repo, http.MethodDelete, "/keys/"+intString(id), nil); err != nil {
						return err
					}
				}
				if wok {
					return e.Mutate(ctx, owner, repo, http.MethodPost, "/keys", map[string]any{"title": name, "key": want.Key, "read_only": want.ReadOnly})
				}
				return nil
			}})
		}
	}
	if d.Environments != nil {
		got := matchNames(*d.Environments, mapValue(s.Additional.Environments))
		for _, name := range unionKeys(*d.Environments, got) {
			want, wok := (*d.Environments)[name]
			cur, cok := got[name]
			if !wok {
				add(p, Change{Remove, "environments." + name, cur, nil, func(ctx context.Context) error {
					return e.Mutate(ctx, owner, repo, http.MethodDelete, "/environments/"+url.PathEscape(name), nil)
				}})
				continue
			}
			before := config.Project(&want, &cur)
			if cok && environmentEqual(want, *before) {
				continue
			}
			// The environment endpoint resets omitted protection settings. Merge the
			// current values before writing, while leaving omitted child collections alone.
			effective := want
			if cok {
				if effective.WaitTimer == nil {
					effective.WaitTimer = cur.WaitTimer
				}
				if effective.PreventSelfReview == nil {
					effective.PreventSelfReview = cur.PreventSelfReview
				}
				if effective.Reviewers == nil {
					effective.Reviewers = cur.Reviewers
				}
				if effective.DeploymentBranchPolicy == nil {
					effective.DeploymentBranchPolicy = cur.DeploymentBranchPolicy
				}
			}
			if effective.DeploymentBranchPolicy == nil && (effective.DeploymentBranchPatterns != nil || effective.DeploymentTagPatterns != nil) {
				effective.DeploymentBranchPolicy = &config.DeploymentBranchPolicy{CustomBranchPolicies: true}
			}
			op := Modify
			if !cok {
				op = Add
			}
			add(p, Change{op, "environments." + name, optional(cok, before), want, func(ctx context.Context) error { return e.SetEnvironment(ctx, owner, repo, name, effective) }})
		}
	}
}
func pagesChanges(p *Plan, owner, repo string, want, got *config.PagesSettings, e Executor) {
	if got == nil {
		got = &config.PagesSettings{}
	}
	exists := got.Enabled != nil && *got.Enabled
	if want.Enabled != nil && !*want.Enabled {
		if exists {
			add(p, Change{Remove, "pages", got, want, func(ctx context.Context) error { return e.Mutate(ctx, owner, repo, http.MethodDelete, "/pages", nil) }})
		}
		return
	}
	if valuesEqual(config.Project(want, got), want) {
		return
	}
	body := object(want)
	delete(body, "enabled")
	if cname, ok := body["cname"].(string); ok && cname == "" {
		body["cname"] = nil
	}
	op := Modify
	if !exists {
		op = Add
	}
	add(p, Change{op, "pages", got, want, func(ctx context.Context) error {
		if !exists {
			create := map[string]any{}
			for _, key := range []string{"build_type", "source"} {
				if v, ok := body[key]; ok {
					create[key] = v
				}
			}
			if len(create) == 0 {
				create["build_type"] = "workflow"
			}
			if err := e.Mutate(ctx, owner, repo, http.MethodPost, "/pages", create); err != nil {
				return err
			}
		}
		if len(body) == 0 {
			return nil
		}
		return e.Mutate(ctx, owner, repo, http.MethodPut, "/pages", body)
	}})
}
func variableChanges(p *Plan, owner, repo, label, path string, want, got *map[string]string, e Executor) {
	if want == nil {
		return
	}
	w, g := map[string]string{}, map[string]string{}
	for k, v := range *want {
		w[strings.ToUpper(k)] = v
	}
	for k, v := range mapValue(got) {
		g[strings.ToUpper(k)] = v
	}
	for _, name := range unionKeys(w, g) {
		value, wok := w[name]
		old, cok := g[name]
		if wok && cok && value == old {
			continue
		}
		op, method, suffix := Add, http.MethodPost, path
		var body any = map[string]string{"name": name, "value": value}
		if cok {
			op, method, suffix = Modify, http.MethodPatch, path+"/"+url.PathEscape(name)
		}
		if !wok {
			op, method, body = Remove, http.MethodDelete, nil
		}
		add(p, Change{op, label + "." + name, optional(cok, old), optional(wok, value), func(ctx context.Context) error { return e.Mutate(ctx, owner, repo, method, suffix, body) }})
	}
}
func object(v any) map[string]any {
	b, _ := json.Marshal(v)
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}
func merged(current, desired any) map[string]any {
	m := object(current)
	maps.Copy(m, object(desired))
	return m
}
func mapValue[T any](m *map[string]T) map[string]T {
	if m == nil {
		return map[string]T{}
	}
	return *m
}
func unionKeys[T any](a, b map[string]T) []string {
	m := map[string]bool{}
	for k := range a {
		m[k] = true
	}
	for k := range b {
		m[k] = true
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func optional(present bool, v any) any {
	if present {
		return v
	}
	return nil
}
func intString(v int64) string { return strconv.FormatInt(v, 10) }

func keyMaterial(v string) string {
	parts := strings.Fields(v)
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, " ")
}
func environmentEqual(a, b config.Environment) bool {
	normalize := func(v config.Environment) config.Environment {
		for _, p := range []**[]string{&v.DeploymentBranchPatterns, &v.DeploymentTagPatterns} {
			if *p != nil {
				items := append([]string{}, (**p)...)
				sort.Strings(items)
				*p = &items
			}
		}
		if v.Reviewers != nil {
			items := append([]config.EnvironmentReviewer{}, (*v.Reviewers)...)
			sort.Slice(items, func(i, j int) bool {
				if items[i].Type != items[j].Type {
					return items[i].Type < items[j].Type
				}
				return items[i].ID < items[j].ID
			})
			v.Reviewers = &items
		}
		if v.Variables != nil {
			items := map[string]string{}
			for k, value := range *v.Variables {
				items[strings.ToUpper(k)] = value
			}
			v.Variables = &items
		}
		return v
	}
	return valuesEqual(normalize(a), normalize(b))
}

// GitHub environment and label names are case-insensitive identities. Match
// existing names before planning so a casing difference never deletes a resource.
func matchNames[T any](want, got map[string]T) map[string]T {
	names := make(map[string]string, len(want))
	for name := range want {
		names[strings.ToLower(name)] = name
	}
	out := make(map[string]T, len(got))
	for name, value := range got {
		if desired, ok := names[strings.ToLower(name)]; ok {
			name = desired
		}
		out[name] = value
	}
	return out
}
