package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	labelColorPattern   = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
	variableNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type CodeScanningSetup struct {
	State       *string   `yaml:"state,omitempty" json:"state,omitempty"`
	Languages   *[]string `yaml:"languages,omitempty" json:"languages,omitempty"`
	QuerySuite  *string   `yaml:"query_suite,omitempty" json:"query_suite,omitempty"`
	ThreatModel *string   `yaml:"threat_model,omitempty" json:"threat_model,omitempty"`
	RunnerType  *string   `yaml:"runner_type,omitempty" json:"runner_type,omitempty"`
	RunnerLabel *string   `yaml:"runner_label,omitempty" json:"runner_label,omitempty"`
}
type PrivateForkWorkflows struct {
	RunWorkflowsFromForkPullRequests  *bool `yaml:"run_workflows_from_fork_pull_requests,omitempty" json:"run_workflows_from_fork_pull_requests,omitempty"`
	SendWriteTokensToWorkflows        *bool `yaml:"send_write_tokens_to_workflows,omitempty" json:"send_write_tokens_to_workflows,omitempty"`
	SendSecretsAndVariables           *bool `yaml:"send_secrets_and_variables,omitempty" json:"send_secrets_and_variables,omitempty"`
	RequireApprovalForForkPRWorkflows *bool `yaml:"require_approval_for_fork_pr_workflows,omitempty" json:"require_approval_for_fork_pr_workflows,omitempty"`
}
type OIDCSettings struct {
	UseDefault          *bool     `yaml:"use_default,omitempty" json:"use_default,omitempty"`
	IncludeClaimKeys    *[]string `yaml:"include_claim_keys,omitempty" json:"include_claim_keys,omitempty"`
	UseImmutableSubject *bool     `yaml:"use_immutable_subject,omitempty" json:"use_immutable_subject,omitempty"`
}
type CacheSettings struct {
	MaxRetentionDays *int `yaml:"max_retention_days,omitempty" json:"max_retention_days,omitempty"`
	MaxSizeGB        *int `yaml:"max_size_gb,omitempty" json:"max_size_gb,omitempty"`
}
type Environment struct {
	WaitTimer                *int                    `yaml:"wait_timer,omitempty" json:"wait_timer,omitempty"`
	PreventSelfReview        *bool                   `yaml:"prevent_self_review,omitempty" json:"prevent_self_review,omitempty"`
	Reviewers                *[]EnvironmentReviewer  `yaml:"reviewers,omitempty" json:"reviewers,omitempty"`
	DeploymentBranchPolicy   *DeploymentBranchPolicy `yaml:"deployment_branch_policy,omitempty" json:"deployment_branch_policy,omitempty"`
	DeploymentBranchPatterns *[]string               `yaml:"deployment_branch_patterns,omitempty" json:"deployment_branch_patterns,omitempty"`
	DeploymentTagPatterns    *[]string               `yaml:"deployment_tag_patterns,omitempty" json:"deployment_tag_patterns,omitempty"`
	Variables                *map[string]string      `yaml:"variables,omitempty" json:"variables,omitempty"`
}
type EnvironmentReviewer struct {
	Type string `yaml:"type" json:"type"`
	ID   int64  `yaml:"id" json:"id"`
}
type DeploymentBranchPolicy struct {
	ProtectedBranches    bool `yaml:"protected_branches" json:"protected_branches"`
	CustomBranchPolicies bool `yaml:"custom_branch_policies" json:"custom_branch_policies"`
}
type PagesSettings struct {
	Enabled       *bool        `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	BuildType     *string      `yaml:"build_type,omitempty" json:"build_type,omitempty"`
	Source        *PagesSource `yaml:"source,omitempty" json:"source,omitempty"`
	CNAME         *string      `yaml:"cname,omitempty" json:"cname,omitempty"`
	HTTPSEnforced *bool        `yaml:"https_enforced,omitempty" json:"https_enforced,omitempty"`
}
type PagesSource struct {
	Branch string `yaml:"branch" json:"branch"`
	Path   string `yaml:"path" json:"path"`
}
type Label struct {
	Color       string `yaml:"color" json:"color"`
	Description string `yaml:"description" json:"description"`
}
type Autolink struct {
	URLTemplate    string `yaml:"url_template" json:"url_template"`
	IsAlphanumeric bool   `yaml:"is_alphanumeric" json:"is_alphanumeric"`
}
type DeployKey struct {
	Key      string `yaml:"key" json:"key"`
	ReadOnly bool   `yaml:"read_only" json:"read_only"`
}

// Project refreshes managed fields from live state. Maps and slices are
// authoritative collections, so their full live contents are retained.
func Project[T any](scope, live *T) *T {
	if scope == nil || live == nil {
		return nil
	}
	result := new(T)
	projectValue(reflect.ValueOf(scope).Elem(), reflect.ValueOf(live).Elem(), reflect.ValueOf(result).Elem())
	return result
}
func projectValue(scope, live, out reflect.Value) {
	if scope.Kind() == reflect.Pointer {
		if scope.IsNil() || live.IsNil() {
			return
		}
		out.Set(reflect.New(scope.Type().Elem()))
		projectValue(scope.Elem(), live.Elem(), out.Elem())
	} else if scope.Kind() == reflect.Struct {
		for i := 0; i < scope.NumField(); i++ {
			projectValue(scope.Field(i), live.Field(i), out.Field(i))
		}
	} else {
		out.Set(live)
	}
}

func (c *Config) validateAdditional() error {
	enum := func(name string, value *string, choices ...string) error {
		if value == nil {
			return nil
		}
		for _, choice := range choices {
			if *value == choice {
				return nil
			}
		}
		return fmt.Errorf("invalid configuration: %s must be one of %s", name, strings.Join(choices, ", "))
	}
	positive := func(name string, value *int) error {
		if value != nil && *value < 1 {
			return fmt.Errorf("invalid configuration: %s must be positive", name)
		}
		return nil
	}
	if a := c.Actions; a != nil {
		if err := enum("actions.fork_pr_contributor_approval", a.ForkPRContributorApproval, "first_time_contributors_new_to_github", "first_time_contributors", "all_external_contributors"); err != nil {
			return err
		}
		if err := enum("actions.access_level", a.AccessLevel, "none", "user", "organization", "enterprise"); err != nil {
			return err
		}
		if err := positive("actions.artifact_and_log_retention_days", a.ArtifactAndLogRetentionDays); err != nil {
			return err
		}
		if a.Cache != nil {
			if err := positive("actions.cache.max_retention_days", a.Cache.MaxRetentionDays); err != nil {
				return err
			}
			if err := positive("actions.cache.max_size_gb", a.Cache.MaxSizeGB); err != nil {
				return err
			}
		}
		if a.OIDC != nil && a.OIDC.UseDefault != nil && *a.OIDC.UseDefault && a.OIDC.IncludeClaimKeys != nil {
			return fmt.Errorf("invalid configuration: actions.oidc.include_claim_keys requires use_default: false")
		}
		if err := validateVariables("actions.variables", a.Variables); err != nil {
			return err
		}
	}
	if c.Security != nil && c.Security.CodeScanningDefaultSetup != nil {
		s := c.Security.CodeScanningDefaultSetup
		if s.State != nil && *s.State == "not-configured" && (s.Languages != nil || s.QuerySuite != nil || s.ThreatModel != nil || s.RunnerType != nil || s.RunnerLabel != nil) {
			return fmt.Errorf("invalid configuration: not-configured code scanning setup cannot include scanner settings")
		}
		for _, item := range []struct {
			name    string
			value   *string
			choices []string
		}{
			{"state", s.State, []string{"configured", "not-configured"}}, {"query_suite", s.QuerySuite, []string{"default", "extended"}}, {"runner_type", s.RunnerType, []string{"standard", "labeled"}}, {"threat_model", s.ThreatModel, []string{"remote", "remote_and_local"}},
		} {
			if err := enum("security.code_scanning_default_setup."+item.name, item.value, item.choices...); err != nil {
				return err
			}
		}
	}
	if p := c.Pages; p != nil {
		if err := enum("pages.build_type", p.BuildType, "legacy", "workflow"); err != nil {
			return err
		}
		if p.Source != nil && p.BuildType != nil && *p.BuildType == "workflow" {
			return fmt.Errorf("invalid configuration: pages.source requires build_type: legacy")
		}
		if p.Source != nil && (p.Source.Branch == "" || (p.Source.Path != "/" && p.Source.Path != "/docs")) {
			return fmt.Errorf("invalid configuration: pages.source requires a branch and path / or /docs")
		}
		if p.Enabled != nil && !*p.Enabled && (p.BuildType != nil || p.Source != nil || p.CNAME != nil || p.HTTPSEnforced != nil) {
			return fmt.Errorf("invalid configuration: pages.enabled: false cannot be combined with site settings")
		}
	}
	if c.Environments != nil {
		if err := validateNames("environments", *c.Environments); err != nil {
			return err
		}
		for name, e := range *c.Environments {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("invalid configuration: empty environment name")
			}
			if e.WaitTimer != nil && (*e.WaitTimer < 0 || *e.WaitTimer > 43200) {
				return fmt.Errorf("invalid configuration: environments.%s.wait_timer must be 0 through 43200", name)
			}
			if e.Reviewers != nil {
				if len(*e.Reviewers) > 6 {
					return fmt.Errorf("invalid configuration: environments.%s supports at most six reviewers", name)
				}
				for _, r := range *e.Reviewers {
					if (r.Type != "User" && r.Type != "Team") || r.ID <= 0 {
						return fmt.Errorf("invalid configuration: environments.%s reviewer requires type User or Team and a positive id", name)
					}
				}
			}
			if p := e.DeploymentBranchPolicy; p != nil && p.ProtectedBranches && p.CustomBranchPolicies {
				return fmt.Errorf("invalid configuration: environments.%s branch policies cannot be both protected and custom", name)
			}
			if (e.DeploymentBranchPatterns != nil && len(*e.DeploymentBranchPatterns) > 0 || e.DeploymentTagPatterns != nil && len(*e.DeploymentTagPatterns) > 0) && e.DeploymentBranchPolicy != nil && !e.DeploymentBranchPolicy.CustomBranchPolicies {
				return fmt.Errorf("invalid configuration: environments.%s patterns require custom_branch_policies", name)
			}
			if err := validateVariables("environments."+name+".variables", e.Variables); err != nil {
				return err
			}
		}
	}
	if c.Labels != nil {
		if err := validateNames("labels", *c.Labels); err != nil {
			return err
		}
		for name, v := range *c.Labels {
			if strings.TrimSpace(name) == "" || !labelColorPattern.MatchString(v.Color) {
				return fmt.Errorf("invalid configuration: labels.%s requires a name and six-digit hexadecimal color", name)
			}
		}
	}
	if c.Autolinks != nil {
		for name, v := range *c.Autolinks {
			if name == "" || !strings.Contains(v.URLTemplate, "<num>") {
				return fmt.Errorf("invalid configuration: autolinks.%s requires a prefix and url_template containing <num>", name)
			}
		}
	}
	if c.DeployKeys != nil {
		for name, v := range *c.DeployKeys {
			if strings.TrimSpace(name) == "" || len(strings.Fields(v.Key)) < 2 {
				return fmt.Errorf("invalid configuration: deploy_keys.%s requires a title and SSH public key", name)
			}
		}
	}
	return nil
}
func validateVariables(path string, variables *map[string]string) error {
	if variables == nil {
		return nil
	}
	seen := map[string]bool{}
	for name := range *variables {
		upper := strings.ToUpper(name)
		if !variableNamePattern.MatchString(name) || strings.HasPrefix(upper, "GITHUB_") || seen[upper] {
			return fmt.Errorf("invalid configuration: %s has invalid or duplicate variable name %q", path, name)
		}
		seen[upper] = true
	}
	return nil
}

func validateNames[T any](section string, values map[string]T) error {
	seen := map[string]bool{}
	for name := range values {
		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("invalid configuration: %s contains names differing only in case", section)
		}
		seen[key] = true
	}
	return nil
}
