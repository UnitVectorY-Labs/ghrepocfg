package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

// Mutate applies a repository-relative configuration operation.
func (c *Client) Mutate(ctx context.Context, owner, repo, method, suffix string, body any) error {
	_, err := c.request(ctx, method, repoPath(owner, repo, suffix), body, nil)
	if err != nil {
		return err
	}
	// GitHub can acknowledge a cache-limit write without exposing the requested
	// effective value. Check these synchronous settings before reporting success.
	if method == http.MethodPut && (suffix == "/actions/cache/retention-limit" || suffix == "/actions/cache/storage-limit") {
		var actual map[string]int
		if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, suffix), nil, &actual); err != nil {
			return fmt.Errorf("verify cache limit after write: %w", err)
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		var expected map[string]int
		if err := json.Unmarshal(encoded, &expected); err != nil {
			return err
		}
		for key, value := range expected {
			if got, ok := actual[key]; !ok || got != value {
				return fmt.Errorf("GitHub accepted %s but read-back %s is %d, requested %d; the requested value is not effective; rerun dry-run to check remaining drift", suffix, key, got, value)
			}
		}
	}
	return nil
}

// wrappedPaged follows pagination on APIs whose arrays are inside an object.
func (c *Client) wrappedPaged(ctx context.Context, path, key string, out any) error {
	next := path + "?per_page=100"
	var all []json.RawMessage
	for next != "" {
		var page map[string]json.RawMessage
		h, err := c.request(ctx, http.MethodGet, next, nil, &page)
		if err != nil {
			return err
		}
		var items []json.RawMessage
		if err := json.Unmarshal(page[key], &items); err != nil {
			return err
		}
		all = append(all, items...)
		next = nextLink(h.Get("Link"))
	}
	b, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
func (c *Client) readVariables(ctx context.Context, path string) (*map[string]string, error) {
	var items []struct{ Name, Value string }
	if err := c.wrappedPaged(ctx, path, "variables", &items); err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, v := range items {
		values[v.Name] = v.Value
	}
	return &values, nil
}
func (c *Client) readAdditional(ctx context.Context, owner, repo string, scope ReadScope, s *State) error {
	if !scope.Full && scope.Desired == nil {
		return nil
	}
	d := scope.Desired
	if d == nil {
		d = &config.Config{}
	}
	read := func(name string, wanted bool, fn func() error) error {
		if !scope.Full && !wanted {
			return nil
		}
		err := fn()
		if scope.Full && (IsStatus(err, 403) || IsStatus(err, 404) || IsStatus(err, 409) || IsStatus(err, 422)) {
			s.Warnings = append(s.Warnings, fmt.Sprintf("%s unavailable; omitted from export: %v", name, err))
			return nil
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		return nil
	}
	get := func(path string, out any) error {
		_, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, path), nil, out)
		return err
	}
	if err := read("repository.immutable_releases", d.Repository != nil && d.Repository.ImmutableReleases != nil, func() error {
		var v struct {
			Enabled bool `json:"enabled"`
		}
		if err := get("/immutable-releases", &v); err != nil {
			return err
		}
		s.Repository.ImmutableReleases = &v.Enabled
		return nil
	}); err != nil {
		return err
	}
	if scope.Full || d.Security != nil {
		if s.Security == nil {
			s.Security = &config.SecuritySettings{}
		}
		wanted := d.Security
		if wanted == nil {
			wanted = &config.SecuritySettings{}
		}
		if err := read("security.private_vulnerability_reporting", wanted.PrivateVulnerabilityReporting != nil, func() error {
			var v struct {
				Enabled bool `json:"enabled"`
			}
			if err := get("/private-vulnerability-reporting", &v); err != nil {
				return err
			}
			s.Security.PrivateVulnerabilityReporting = &v.Enabled
			return nil
		}); err != nil {
			return err
		}
		if err := read("security.code_scanning_default_setup", wanted.CodeScanningDefaultSetup != nil, func() error {
			var v config.CodeScanningSetup
			if err := get("/code-scanning/default-setup", &v); err != nil {
				return err
			}
			if v.State != nil && *v.State == "not-configured" {
				v = config.CodeScanningSetup{State: v.State}
			}
			s.Security.CodeScanningDefaultSetup = &v
			return nil
		}); err != nil {
			return err
		}
	}
	if scope.Full || d.Actions != nil {
		if s.Actions == nil {
			s.Actions = &config.ActionsSettings{}
		}
		a := d.Actions
		if a == nil {
			a = &config.ActionsSettings{}
		}
		entries := []struct {
			name   string
			wanted bool
			fn     func() error
		}{
			{"artifact_and_log_retention_days", a.ArtifactAndLogRetentionDays != nil, func() error {
				var v struct{ Days int }
				if err := get("/actions/permissions/artifact-and-log-retention", &v); err != nil {
					return err
				}
				s.Actions.ArtifactAndLogRetentionDays = &v.Days
				return nil
			}},
			{"fork_pr_contributor_approval", a.ForkPRContributorApproval != nil, func() error {
				var v struct {
					Policy string `json:"approval_policy"`
				}
				if err := get("/actions/permissions/fork-pr-contributor-approval", &v); err != nil {
					return err
				}
				s.Actions.ForkPRContributorApproval = &v.Policy
				return nil
			}},
			{"private_fork_workflows", a.PrivateForkWorkflows != nil, func() error {
				var v config.PrivateForkWorkflows
				if err := get("/actions/permissions/fork-pr-workflows-private-repos", &v); err != nil {
					return err
				}
				s.Actions.PrivateForkWorkflows = &v
				return nil
			}},
			{"access_level", a.AccessLevel != nil, func() error {
				var v struct {
					Level string `json:"access_level"`
				}
				if err := get("/actions/permissions/access", &v); err != nil {
					return err
				}
				s.Actions.AccessLevel = &v.Level
				return nil
			}},
			{"oidc", a.OIDC != nil, func() error {
				var v config.OIDCSettings
				if err := get("/actions/oidc/customization/sub", &v); err != nil {
					return err
				}
				if v.UseDefault != nil && *v.UseDefault {
					v.IncludeClaimKeys = nil
				}
				s.Actions.OIDC = &v
				return nil
			}},
			{"cache.max_retention_days", a.Cache != nil && a.Cache.MaxRetentionDays != nil, func() error {
				var v struct {
					Days int `json:"max_cache_retention_days"`
				}
				if err := get("/actions/cache/retention-limit", &v); err != nil {
					return err
				}
				if s.Actions.Cache == nil {
					s.Actions.Cache = &config.CacheSettings{}
				}
				s.Actions.Cache.MaxRetentionDays = &v.Days
				return nil
			}},
			{"cache.max_size_gb", a.Cache != nil && a.Cache.MaxSizeGB != nil, func() error {
				var v struct {
					Size int `json:"max_cache_size_gb"`
				}
				if err := get("/actions/cache/storage-limit", &v); err != nil {
					return err
				}
				if s.Actions.Cache == nil {
					s.Actions.Cache = &config.CacheSettings{}
				}
				s.Actions.Cache.MaxSizeGB = &v.Size
				return nil
			}},
			{"variables", a.Variables != nil, func() error {
				v, err := c.readVariables(ctx, repoPath(owner, repo, "/actions/variables"))
				if err != nil {
					return err
				}
				s.Actions.Variables = v
				return nil
			}},
		}
		for _, entry := range entries {
			if err := read("actions."+entry.name, entry.wanted, entry.fn); err != nil {
				return err
			}
		}
	}
	if err := read("pages", d.Pages != nil, func() error {
		var v config.PagesSettings
		err := get("/pages", &v)
		if IsStatus(err, 404) {
			disabled := false
			s.Additional.Pages = &config.PagesSettings{Enabled: &disabled}
			return nil
		}
		if err != nil {
			return err
		}
		enabled := true
		v.Enabled = &enabled
		if v.CNAME == nil {
			empty := ""
			v.CNAME = &empty
		}
		if v.BuildType != nil && *v.BuildType == "workflow" {
			v.Source = nil
		}
		s.Additional.Pages = &v
		return nil
	}); err != nil {
		return err
	}
	if err := read("labels", d.Labels != nil, func() error {
		var items []struct {
			Name        string
			Color       string
			Description string
		}
		if err := c.paged(ctx, repoPath(owner, repo, "/labels"), &items); err != nil {
			return err
		}
		v := map[string]config.Label{}
		for _, item := range items {
			v[item.Name] = config.Label{Color: item.Color, Description: item.Description}
		}
		s.Additional.Labels = &v
		return nil
	}); err != nil {
		return err
	}
	if err := read("autolinks", d.Autolinks != nil, func() error {
		var items []struct {
			ID     int64
			Prefix string `json:"key_prefix"`
			config.Autolink
		}
		if err := c.paged(ctx, repoPath(owner, repo, "/autolinks"), &items); err != nil {
			return err
		}
		v := map[string]config.Autolink{}
		s.AutolinkIDs = map[string]int64{}
		for _, item := range items {
			if _, ok := v[item.Prefix]; ok {
				return fmt.Errorf("duplicate autolink prefix %q", item.Prefix)
			}
			v[item.Prefix] = item.Autolink
			s.AutolinkIDs[item.Prefix] = item.ID
		}
		s.Additional.Autolinks = &v
		return nil
	}); err != nil {
		return err
	}
	if err := read("deploy_keys", d.DeployKeys != nil, func() error {
		var items []struct {
			ID    int64
			Title string
			config.DeployKey
		}
		if err := c.paged(ctx, repoPath(owner, repo, "/keys"), &items); err != nil {
			return err
		}
		v := map[string]config.DeployKey{}
		s.DeployKeyIDs = map[string]int64{}
		for _, item := range items {
			if _, ok := v[item.Title]; ok {
				return fmt.Errorf("duplicate deploy key title %q", item.Title)
			}
			v[item.Title] = item.DeployKey
			s.DeployKeyIDs[item.Title] = item.ID
		}
		s.Additional.DeployKeys = &v
		return nil
	}); err != nil {
		return err
	}
	return read("environments", d.Environments != nil, func() error {
		var items []struct{ Name string }
		if err := c.wrappedPaged(ctx, repoPath(owner, repo, "/environments"), "environments", &items); err != nil {
			return err
		}
		v := map[string]config.Environment{}
		for _, item := range items {
			e, err := c.readEnvironment(ctx, owner, repo, item.Name)
			if err != nil {
				return err
			}
			v[item.Name] = e
		}
		s.Additional.Environments = &v
		return nil
	})
}
func (c *Client) readEnvironment(ctx context.Context, owner, repo, name string) (config.Environment, error) {
	path := repoPath(owner, repo, "/environments/"+url.PathEscape(name))
	var raw struct {
		DeploymentBranchPolicy *config.DeploymentBranchPolicy `json:"deployment_branch_policy"`
		Rules                  []struct {
			Type              string
			WaitTimer         int  `json:"wait_timer"`
			PreventSelfReview bool `json:"prevent_self_review"`
			Reviewers         []struct {
				Type     string
				Reviewer struct{ ID int64 }
			}
		} `json:"protection_rules"`
	}
	if _, err := c.request(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return config.Environment{}, err
	}
	zero, no := 0, false
	reviewers := []config.EnvironmentReviewer{}
	e := config.Environment{WaitTimer: &zero, PreventSelfReview: &no, Reviewers: &reviewers, DeploymentBranchPolicy: raw.DeploymentBranchPolicy}
	if e.DeploymentBranchPolicy == nil {
		e.DeploymentBranchPolicy = &config.DeploymentBranchPolicy{}
	}
	for _, r := range raw.Rules {
		switch r.Type {
		case "wait_timer":
			e.WaitTimer = &r.WaitTimer
		case "required_reviewers":
			e.PreventSelfReview = &r.PreventSelfReview
			for _, v := range r.Reviewers {
				reviewers = append(reviewers, config.EnvironmentReviewer{Type: v.Type, ID: v.Reviewer.ID})
			}
		}
	}
	branches, tags := []string{}, []string{}
	if raw.DeploymentBranchPolicy != nil && raw.DeploymentBranchPolicy.CustomBranchPolicies {
		var policies []struct{ Name, Type string }
		if err := c.wrappedPaged(ctx, path+"/deployment-branch-policies", "branch_policies", &policies); err != nil {
			return e, err
		}
		for _, p := range policies {
			if p.Type == "tag" {
				tags = append(tags, p.Name)
			} else {
				branches = append(branches, p.Name)
			}
		}
	}
	e.DeploymentBranchPatterns = &branches
	e.DeploymentTagPatterns = &tags
	variables, err := c.readVariables(ctx, path+"/variables")
	if err != nil {
		return e, err
	}
	e.Variables = variables
	return e, nil
}

// SetEnvironment preserves omitted settings and changes only requested policy
// collections. Creation precedes dependent variables and branch/tag policies.
func (c *Client) SetEnvironment(ctx context.Context, owner, repo, name string, want config.Environment) error {
	if policy := want.DeploymentBranchPolicy; policy != nil && !policy.CustomBranchPolicies {
		if (want.DeploymentBranchPatterns != nil && len(*want.DeploymentBranchPatterns) > 0) || (want.DeploymentTagPatterns != nil && len(*want.DeploymentTagPatterns) > 0) {
			return fmt.Errorf("environment %s requires deployment_branch_policy.custom_branch_policies: true before applying branch/tag patterns", name)
		}
	}
	suffix := "/environments/" + url.PathEscape(name)
	body := map[string]any{}
	if want.WaitTimer != nil {
		body["wait_timer"] = *want.WaitTimer
	}
	if want.PreventSelfReview != nil {
		body["prevent_self_review"] = *want.PreventSelfReview
	}
	if want.Reviewers != nil {
		body["reviewers"] = *want.Reviewers
	}
	if p := want.DeploymentBranchPolicy; p != nil {
		if !p.ProtectedBranches && !p.CustomBranchPolicies {
			body["deployment_branch_policy"] = nil
		} else {
			body["deployment_branch_policy"] = p
		}
	}
	if err := c.Mutate(ctx, owner, repo, http.MethodPut, suffix, body); err != nil {
		return err
	}
	if want.DeploymentBranchPolicy == nil || want.DeploymentBranchPolicy.CustomBranchPolicies {
		if want.DeploymentBranchPatterns != nil || want.DeploymentTagPatterns != nil {
			var policies []struct {
				ID         int64
				Name, Type string
			}
			if err := c.wrappedPaged(ctx, repoPath(owner, repo, suffix+"/deployment-branch-policies"), "branch_policies", &policies); err != nil {
				return err
			}
			for kind, patterns := range map[string]*[]string{"branch": want.DeploymentBranchPatterns, "tag": want.DeploymentTagPatterns} {
				if patterns == nil {
					continue
				}
				desired := map[string]bool{}
				for _, pattern := range *patterns {
					desired[pattern] = true
				}
				for _, p := range policies {
					if p.Type != kind {
						continue
					}
					if desired[p.Name] {
						delete(desired, p.Name)
					} else if err := c.Mutate(ctx, owner, repo, http.MethodDelete, suffix+"/deployment-branch-policies/"+intPath(p.ID), nil); err != nil {
						return err
					}
				}
				for pattern := range desired {
					if err := c.Mutate(ctx, owner, repo, http.MethodPost, suffix+"/deployment-branch-policies", map[string]string{"name": pattern, "type": kind}); err != nil {
						return err
					}
				}
			}
		}
	}
	if want.Variables != nil {
		current, err := c.readVariables(ctx, repoPath(owner, repo, suffix+"/variables"))
		if err != nil {
			return err
		}
		return c.replaceVariables(ctx, owner, repo, suffix+"/variables", *want.Variables, *current)
	}
	return nil
}
func (c *Client) replaceVariables(ctx context.Context, owner, repo, path string, want, got map[string]string) error {
	normalized := map[string]string{}
	for name, value := range want {
		normalized[strings.ToUpper(name)] = value
	}
	for name := range got {
		if _, ok := normalized[strings.ToUpper(name)]; !ok {
			if err := c.Mutate(ctx, owner, repo, http.MethodDelete, path+"/"+url.PathEscape(name), nil); err != nil {
				return err
			}
		}
	}
	for name, value := range normalized {
		old, exists := got[name]
		if exists && old == value {
			continue
		}
		method, suffix := http.MethodPost, path
		if exists {
			method, suffix = http.MethodPatch, path+"/"+url.PathEscape(name)
		}
		if err := c.Mutate(ctx, owner, repo, method, suffix, map[string]string{"name": name, "value": value}); err != nil {
			return err
		}
	}
	return nil
}
