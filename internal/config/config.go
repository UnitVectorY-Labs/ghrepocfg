// Package config defines and validates ghrepocfg's literal YAML format.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the complete v1 configuration. A nil section is unmanaged, while
// a present collection (including an empty one) is authoritative.
type Config struct {
	Repository    *RepositorySettings `yaml:"repository,omitempty" json:"repository,omitempty"`
	Security      *SecuritySettings   `yaml:"security,omitempty" json:"security,omitempty"`
	Actions       *ActionsSettings    `yaml:"actions,omitempty" json:"actions,omitempty"`
	Collaborators *map[string]Access  `yaml:"collaborators,omitempty" json:"collaborators,omitempty"`
	Teams         *map[string]Access  `yaml:"teams,omitempty" json:"teams,omitempty"`
	Rulesets      *map[string]Ruleset `yaml:"rulesets,omitempty" json:"rulesets,omitempty"`
}

// RepositorySettings contains safe, mutable fields accepted by GitHub's
// Update a repository endpoint. Identity, visibility, archive state, and other
// destructive fields are intentionally absent.
type RepositorySettings struct {
	Description               *string   `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage                  *string   `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	HasIssues                 *bool     `yaml:"has_issues,omitempty" json:"has_issues,omitempty"`
	HasProjects               *bool     `yaml:"has_projects,omitempty" json:"has_projects,omitempty"`
	HasWiki                   *bool     `yaml:"has_wiki,omitempty" json:"has_wiki,omitempty"`
	HasDiscussions            *bool     `yaml:"has_discussions,omitempty" json:"has_discussions,omitempty"`
	HasPullRequests           *bool     `yaml:"has_pull_requests,omitempty" json:"has_pull_requests,omitempty"`
	PullRequestCreationPolicy *string   `yaml:"pull_request_creation_policy,omitempty" json:"pull_request_creation_policy,omitempty"`
	IsTemplate                *bool     `yaml:"is_template,omitempty" json:"is_template,omitempty"`
	DefaultBranch             *string   `yaml:"default_branch,omitempty" json:"default_branch,omitempty"`
	AllowSquashMerge          *bool     `yaml:"allow_squash_merge,omitempty" json:"allow_squash_merge,omitempty"`
	AllowMergeCommit          *bool     `yaml:"allow_merge_commit,omitempty" json:"allow_merge_commit,omitempty"`
	AllowRebaseMerge          *bool     `yaml:"allow_rebase_merge,omitempty" json:"allow_rebase_merge,omitempty"`
	AllowAutoMerge            *bool     `yaml:"allow_auto_merge,omitempty" json:"allow_auto_merge,omitempty"`
	DeleteBranchOnMerge       *bool     `yaml:"delete_branch_on_merge,omitempty" json:"delete_branch_on_merge,omitempty"`
	AllowUpdateBranch         *bool     `yaml:"allow_update_branch,omitempty" json:"allow_update_branch,omitempty"`
	UseSquashPRTitleAsDefault *bool     `yaml:"use_squash_pr_title_as_default,omitempty" json:"use_squash_pr_title_as_default,omitempty"`
	SquashMergeCommitTitle    *string   `yaml:"squash_merge_commit_title,omitempty" json:"squash_merge_commit_title,omitempty"`
	SquashMergeCommitMessage  *string   `yaml:"squash_merge_commit_message,omitempty" json:"squash_merge_commit_message,omitempty"`
	MergeCommitTitle          *string   `yaml:"merge_commit_title,omitempty" json:"merge_commit_title,omitempty"`
	MergeCommitMessage        *string   `yaml:"merge_commit_message,omitempty" json:"merge_commit_message,omitempty"`
	WebCommitSignoffRequired  *bool     `yaml:"web_commit_signoff_required,omitempty" json:"web_commit_signoff_required,omitempty"`
	AllowForking              *bool     `yaml:"allow_forking,omitempty" json:"allow_forking,omitempty"`
	Topics                    *[]string `yaml:"topics,omitempty" json:"topics,omitempty"`
}

type SecuritySettings struct {
	VulnerabilityAlerts                   *bool                   `yaml:"vulnerability_alerts,omitempty" json:"vulnerability_alerts,omitempty"`
	AutomatedSecurityFixes                *bool                   `yaml:"automated_security_fixes,omitempty" json:"automated_security_fixes,omitempty"`
	AdvancedSecurity                      *FeatureStatus          `yaml:"advanced_security,omitempty" json:"advanced_security,omitempty"`
	CodeSecurity                          *FeatureStatus          `yaml:"code_security,omitempty" json:"code_security,omitempty"`
	SecretScanning                        *FeatureStatus          `yaml:"secret_scanning,omitempty" json:"secret_scanning,omitempty"`
	SecretScanningPushProtection          *FeatureStatus          `yaml:"secret_scanning_push_protection,omitempty" json:"secret_scanning_push_protection,omitempty"`
	SecretScanningAIDetection             *FeatureStatus          `yaml:"secret_scanning_ai_detection,omitempty" json:"secret_scanning_ai_detection,omitempty"`
	SecretScanningNonProviderPatterns     *FeatureStatus          `yaml:"secret_scanning_non_provider_patterns,omitempty" json:"secret_scanning_non_provider_patterns,omitempty"`
	SecretScanningDelegatedAlertDismissal *FeatureStatus          `yaml:"secret_scanning_delegated_alert_dismissal,omitempty" json:"secret_scanning_delegated_alert_dismissal,omitempty"`
	SecretScanningDelegatedBypass         *FeatureStatus          `yaml:"secret_scanning_delegated_bypass,omitempty" json:"secret_scanning_delegated_bypass,omitempty"`
	SecretScanningDelegatedBypassOptions  *DelegatedBypassOptions `yaml:"secret_scanning_delegated_bypass_options,omitempty" json:"secret_scanning_delegated_bypass_options,omitempty"`
}

type FeatureStatus struct {
	Status string `yaml:"status" json:"status"`
}

type DelegatedBypassOptions struct {
	Reviewers []BypassReviewer `yaml:"reviewers" json:"reviewers"`
}

type BypassReviewer struct {
	ReviewerID   int64  `yaml:"reviewer_id" json:"reviewer_id"`
	ReviewerType string `yaml:"reviewer_type" json:"reviewer_type"`
	Mode         string `yaml:"mode,omitempty" json:"mode,omitempty"`
}

type ActionsSettings struct {
	Enabled                      *bool            `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	AllowedActions               *string          `yaml:"allowed_actions,omitempty" json:"allowed_actions,omitempty"`
	SelectedActions              *SelectedActions `yaml:"selected_actions,omitempty" json:"selected_actions,omitempty"`
	DefaultWorkflowPermissions   *string          `yaml:"default_workflow_permissions,omitempty" json:"default_workflow_permissions,omitempty"`
	CanApprovePullRequestReviews *bool            `yaml:"can_approve_pull_request_reviews,omitempty" json:"can_approve_pull_request_reviews,omitempty"`
}

type SelectedActions struct {
	GitHubOwnedAllowed *bool     `yaml:"github_owned_allowed,omitempty" json:"github_owned_allowed,omitempty"`
	VerifiedAllowed    *bool     `yaml:"verified_allowed,omitempty" json:"verified_allowed,omitempty"`
	PatternsAllowed    *[]string `yaml:"patterns_allowed,omitempty" json:"patterns_allowed,omitempty"`
}

type Access struct {
	Permission string `yaml:"permission" json:"permission"`
}

type Ruleset struct {
	Target       string        `yaml:"target,omitempty" json:"target,omitempty"`
	Enforcement  string        `yaml:"enforcement" json:"enforcement"`
	BypassActors []BypassActor `yaml:"bypass_actors,omitempty" json:"bypass_actors,omitempty"`
	Conditions   Conditions    `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	Rules        []Rule        `yaml:"rules,omitempty" json:"rules,omitempty"`
}

type BypassActor struct {
	ActorID    *int64 `yaml:"actor_id,omitempty" json:"actor_id,omitempty"`
	ActorType  string `yaml:"actor_type" json:"actor_type"`
	BypassMode string `yaml:"bypass_mode,omitempty" json:"bypass_mode,omitempty"`
}

type Conditions struct {
	RefName *RefNameCondition `yaml:"ref_name,omitempty" json:"ref_name,omitempty"`
}

type RefNameCondition struct {
	Include []string `yaml:"include" json:"include"`
	Exclude []string `yaml:"exclude" json:"exclude"`
}

type Rule struct {
	Type       string          `yaml:"type" json:"type"`
	Parameters *RuleParameters `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// RuleParameters is the union of the documented repository ruleset parameter
// objects. KnownFields decoding makes misspelled and future unsupported keys fail.
type RuleParameters struct {
	UpdateAllowsFetchAndMerge        *bool                       `yaml:"update_allows_fetch_and_merge,omitempty" json:"update_allows_fetch_and_merge,omitempty"`
	CheckResponseTimeoutMinutes      *int                        `yaml:"check_response_timeout_minutes,omitempty" json:"check_response_timeout_minutes,omitempty"`
	GroupingStrategy                 *string                     `yaml:"grouping_strategy,omitempty" json:"grouping_strategy,omitempty"`
	MaxEntriesToBuild                *int                        `yaml:"max_entries_to_build,omitempty" json:"max_entries_to_build,omitempty"`
	MaxEntriesToMerge                *int                        `yaml:"max_entries_to_merge,omitempty" json:"max_entries_to_merge,omitempty"`
	MergeMethod                      *string                     `yaml:"merge_method,omitempty" json:"merge_method,omitempty"`
	MinEntriesToMerge                *int                        `yaml:"min_entries_to_merge,omitempty" json:"min_entries_to_merge,omitempty"`
	MinEntriesToMergeWaitMinutes     *int                        `yaml:"min_entries_to_merge_wait_minutes,omitempty" json:"min_entries_to_merge_wait_minutes,omitempty"`
	DismissStaleReviewsOnPush        *bool                       `yaml:"dismiss_stale_reviews_on_push,omitempty" json:"dismiss_stale_reviews_on_push,omitempty"`
	DismissalRestriction             *DismissalRestriction       `yaml:"dismissal_restriction,omitempty" json:"dismissal_restriction,omitempty"`
	RequireCodeOwnerReview           *bool                       `yaml:"require_code_owner_review,omitempty" json:"require_code_owner_review,omitempty"`
	RequireLastPushApproval          *bool                       `yaml:"require_last_push_approval,omitempty" json:"require_last_push_approval,omitempty"`
	RequiredApprovingReviewCount     *int                        `yaml:"required_approving_review_count,omitempty" json:"required_approving_review_count,omitempty"`
	RequiredReviewThreadResolution   *bool                       `yaml:"required_review_thread_resolution,omitempty" json:"required_review_thread_resolution,omitempty"`
	RequiredReviewers                *[]RequiredReviewer         `yaml:"required_reviewers,omitempty" json:"required_reviewers,omitempty"`
	AllowedMergeMethods              *[]string                   `yaml:"allowed_merge_methods,omitempty" json:"allowed_merge_methods,omitempty"`
	RequiredStatusChecks             *[]RequiredStatusCheck      `yaml:"required_status_checks,omitempty" json:"required_status_checks,omitempty"`
	StrictRequiredStatusChecksPolicy *bool                       `yaml:"strict_required_status_checks_policy,omitempty" json:"strict_required_status_checks_policy,omitempty"`
	DoNotEnforceOnCreate             *bool                       `yaml:"do_not_enforce_on_create,omitempty" json:"do_not_enforce_on_create,omitempty"`
	RequiredDeploymentEnvironments   *[]string                   `yaml:"required_deployment_environments,omitempty" json:"required_deployment_environments,omitempty"`
	Name                             *string                     `yaml:"name,omitempty" json:"name,omitempty"`
	Negate                           *bool                       `yaml:"negate,omitempty" json:"negate,omitempty"`
	Operator                         *string                     `yaml:"operator,omitempty" json:"operator,omitempty"`
	Pattern                          *string                     `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	RestrictedFilePaths              *[]string                   `yaml:"restricted_file_paths,omitempty" json:"restricted_file_paths,omitempty"`
	MaxFilePathLength                *int                        `yaml:"max_file_path_length,omitempty" json:"max_file_path_length,omitempty"`
	RestrictedFileExtensions         *[]string                   `yaml:"restricted_file_extensions,omitempty" json:"restricted_file_extensions,omitempty"`
	MaxFileSize                      *int                        `yaml:"max_file_size,omitempty" json:"max_file_size,omitempty"`
	Workflows                        *[]RequiredWorkflow         `yaml:"workflows,omitempty" json:"workflows,omitempty"`
	CodeScanningTools                *[]RequiredCodeScanningTool `yaml:"code_scanning_tools,omitempty" json:"code_scanning_tools,omitempty"`
	ReviewDraftPullRequests          *bool                       `yaml:"review_draft_pull_requests,omitempty" json:"review_draft_pull_requests,omitempty"`
	ReviewOnPush                     *bool                       `yaml:"review_on_push,omitempty" json:"review_on_push,omitempty"`
}

type DismissalRestriction struct {
	AllowedActors []RuleActor `yaml:"allowed_actors" json:"allowed_actors"`
}

type RuleActor struct {
	ID   int64  `yaml:"id" json:"id"`
	Type string `yaml:"type" json:"type"`
}

type RequiredReviewer struct {
	FilePatterns     []string  `yaml:"file_patterns" json:"file_patterns"`
	MinimumApprovals int       `yaml:"minimum_approvals" json:"minimum_approvals"`
	Reviewer         RuleActor `yaml:"reviewer" json:"reviewer"`
}

type RequiredStatusCheck struct {
	Context       string `yaml:"context" json:"context"`
	IntegrationID *int64 `yaml:"integration_id,omitempty" json:"integration_id,omitempty"`
}

type RequiredWorkflow struct {
	Path         string  `yaml:"path" json:"path"`
	RepositoryID int64   `yaml:"repository_id" json:"repository_id"`
	Ref          *string `yaml:"ref,omitempty" json:"ref,omitempty"`
	SHA          *string `yaml:"sha,omitempty" json:"sha,omitempty"`
}

type RequiredCodeScanningTool struct {
	Tool                    string `yaml:"tool" json:"tool"`
	SecurityAlertsThreshold string `yaml:"security_alerts_threshold" json:"security_alerts_threshold"`
	AlertsThreshold         string `yaml:"alerts_threshold" json:"alerts_threshold"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

func Parse(b []byte) (*Config, error) {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return nil, errors.New("invalid configuration: JSON configuration is not supported; use YAML block syntax")
	}
	if err := rejectIntentionallyUnmanaged(b); err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid configuration: multiple YAML documents are not supported")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func rejectIntentionallyUnmanaged(b []byte) error {
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil
	}
	if len(root.Content) == 0 {
		return nil
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		key, val := doc.Content[i].Value, doc.Content[i+1]
		if key == "secrets" {
			return errors.New("invalid configuration: secrets are intentionally unmanaged because their values cannot be read from GitHub")
		}
		if key != "repository" || val.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(val.Content); j += 2 {
			k := val.Content[j].Value
			switch k {
			case "visibility", "private":
				return fmt.Errorf("invalid configuration: repository.%s is intentionally unmanaged because visibility changes are high risk", k)
			case "archived":
				return errors.New("invalid configuration: repository.archived is intentionally unmanaged because archive changes are high risk")
			case "name", "owner":
				return fmt.Errorf("invalid configuration: repository.%s is identity and intentionally unmanaged", k)
			}
		}
	}
	return nil
}

func (c *Config) Validate() error {
	if c.Repository == nil && c.Security == nil && c.Actions == nil && c.Collaborators == nil && c.Teams == nil && c.Rulesets == nil {
		return errors.New("invalid configuration: at least one managed section is required")
	}
	validPermission := func(p string) bool {
		return p == "pull" || p == "triage" || p == "push" || p == "maintain" || p == "admin" || strings.HasPrefix(p, "custom:")
	}
	for section, entries := range map[string]*map[string]Access{"collaborators": c.Collaborators, "teams": c.Teams} {
		if entries == nil {
			continue
		}
		seen := map[string]string{}
		for name, access := range *entries {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("invalid configuration: %s contains an empty name", section)
			}
			if !validPermission(access.Permission) {
				return fmt.Errorf("invalid configuration: %s.%s.permission %q is not pull, triage, push, maintain, admin, or custom:ROLE", section, name, access.Permission)
			}
			lower := strings.ToLower(name)
			if prior, ok := seen[lower]; ok {
				return fmt.Errorf("invalid configuration: %s names %q and %q differ only by case", section, prior, name)
			}
			seen[lower] = name
		}
	}
	if c.Actions != nil {
		if c.Actions.AllowedActions != nil && *c.Actions.AllowedActions != "all" && *c.Actions.AllowedActions != "local_only" && *c.Actions.AllowedActions != "selected" {
			return fmt.Errorf("invalid configuration: actions.allowed_actions must be all, local_only, or selected")
		}
		if c.Actions.DefaultWorkflowPermissions != nil && *c.Actions.DefaultWorkflowPermissions != "read" && *c.Actions.DefaultWorkflowPermissions != "write" {
			return fmt.Errorf("invalid configuration: actions.default_workflow_permissions must be read or write")
		}
	}
	if c.Repository != nil && c.Repository.PullRequestCreationPolicy != nil && *c.Repository.PullRequestCreationPolicy != "all" && *c.Repository.PullRequestCreationPolicy != "collaborators_only" {
		return errors.New("invalid configuration: repository.pull_request_creation_policy must be all or collaborators_only")
	}
	if c.Repository != nil {
		enums := []struct {
			name  string
			value *string
			valid map[string]bool
		}{{"squash_merge_commit_title", c.Repository.SquashMergeCommitTitle, map[string]bool{"PR_TITLE": true, "COMMIT_OR_PR_TITLE": true}}, {"squash_merge_commit_message", c.Repository.SquashMergeCommitMessage, map[string]bool{"PR_BODY": true, "COMMIT_MESSAGES": true, "BLANK": true}}, {"merge_commit_title", c.Repository.MergeCommitTitle, map[string]bool{"PR_TITLE": true, "MERGE_MESSAGE": true}}, {"merge_commit_message", c.Repository.MergeCommitMessage, map[string]bool{"PR_TITLE": true, "PR_BODY": true, "BLANK": true}}}
		for _, item := range enums {
			if item.value != nil && !item.valid[*item.value] {
				return fmt.Errorf("invalid configuration: repository.%s has unsupported value %q", item.name, *item.value)
			}
		}
	}
	if c.Security != nil {
		features := map[string]*FeatureStatus{"advanced_security": c.Security.AdvancedSecurity, "code_security": c.Security.CodeSecurity, "secret_scanning": c.Security.SecretScanning, "secret_scanning_push_protection": c.Security.SecretScanningPushProtection, "secret_scanning_ai_detection": c.Security.SecretScanningAIDetection, "secret_scanning_non_provider_patterns": c.Security.SecretScanningNonProviderPatterns, "secret_scanning_delegated_alert_dismissal": c.Security.SecretScanningDelegatedAlertDismissal, "secret_scanning_delegated_bypass": c.Security.SecretScanningDelegatedBypass}
		for name, v := range features {
			if v != nil && v.Status != "enabled" && v.Status != "disabled" {
				return fmt.Errorf("invalid configuration: security.%s.status must be enabled or disabled", name)
			}
		}
	}
	if c.Rulesets != nil {
		for name, rs := range *c.Rulesets {
			if strings.TrimSpace(name) == "" {
				return errors.New("invalid configuration: rulesets contains an empty name")
			}
			if rs.Target != "" && rs.Target != "branch" && rs.Target != "tag" && rs.Target != "push" {
				return fmt.Errorf("invalid configuration: rulesets.%s.target must be branch, tag, or push", name)
			}
			if rs.Enforcement != "active" && rs.Enforcement != "disabled" && rs.Enforcement != "evaluate" {
				return fmt.Errorf("invalid configuration: rulesets.%s.enforcement must be active, disabled, or evaluate", name)
			}
			for i, rule := range rs.Rules {
				if strings.TrimSpace(rule.Type) == "" {
					return fmt.Errorf("invalid configuration: rulesets.%s.rules[%d].type is required", name, i)
				}
				if err := validateRule(rule); err != nil {
					return fmt.Errorf("invalid configuration: rulesets.%s.rules[%d]: %w", name, i, err)
				}
			}
			for i, a := range rs.BypassActors {
				validActor := map[string]bool{"Integration": true, "OrganizationAdmin": true, "RepositoryRole": true, "Team": true, "DeployKey": true, "User": true}
				if !validActor[a.ActorType] {
					return fmt.Errorf("invalid configuration: rulesets.%s.bypass_actors[%d].actor_type is invalid", name, i)
				}
				if a.BypassMode != "" && a.BypassMode != "always" && a.BypassMode != "pull_request" && a.BypassMode != "exempt" {
					return fmt.Errorf("invalid configuration: rulesets.%s.bypass_actors[%d].bypass_mode is invalid", name, i)
				}
			}
		}
	}
	return nil
}

func validateRule(rule Rule) error {
	allowed := map[string][]string{
		"creation": {}, "deletion": {}, "required_linear_history": {}, "required_signatures": {}, "non_fast_forward": {}, "license_compliance_scanning": {},
		"update": {"update_allows_fetch_and_merge"}, "merge_queue": {"check_response_timeout_minutes", "grouping_strategy", "max_entries_to_build", "max_entries_to_merge", "merge_method", "min_entries_to_merge", "min_entries_to_merge_wait_minutes"}, "required_deployments": {"required_deployment_environments"},
		"pull_request":           {"allowed_merge_methods", "dismiss_stale_reviews_on_push", "dismissal_restriction", "require_code_owner_review", "require_last_push_approval", "required_approving_review_count", "required_review_thread_resolution", "required_reviewers"},
		"required_status_checks": {"required_status_checks", "strict_required_status_checks_policy", "do_not_enforce_on_create"},
		"commit_message_pattern": {"name", "negate", "operator", "pattern"}, "commit_author_email_pattern": {"name", "negate", "operator", "pattern"}, "committer_email_pattern": {"name", "negate", "operator", "pattern"}, "branch_name_pattern": {"name", "negate", "operator", "pattern"}, "tag_name_pattern": {"name", "negate", "operator", "pattern"},
		"workflows": {"do_not_enforce_on_create", "workflows"}, "code_scanning": {"code_scanning_tools"}, "copilot_code_review": {"review_draft_pull_requests", "review_on_push"}, "file_path_restriction": {"restricted_file_paths"}, "max_file_path_length": {"max_file_path_length"}, "file_extension_restriction": {"restricted_file_extensions"}, "max_file_size": {"max_file_size"},
	}
	fields, ok := allowed[rule.Type]
	if !ok {
		return fmt.Errorf("unsupported rule type %q", rule.Type)
	}
	allowedField := map[string]bool{}
	for _, f := range fields {
		allowedField[f] = true
	}
	if rule.Parameters == nil {
		return nil
	}
	if len(fields) == 0 {
		return fmt.Errorf("rule type %q does not accept parameters", rule.Type)
	}
	rv := reflect.ValueOf(rule.Parameters).Elem()
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		if rv.Field(i).IsNil() {
			continue
		}
		name := strings.Split(rt.Field(i).Tag.Get("json"), ",")[0]
		if !allowedField[name] {
			return fmt.Errorf("parameter %q is not valid for rule type %q", name, rule.Type)
		}
	}
	return nil
}

func Marshal(c *Config) ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return append([]byte("# Generated by ghrepocfg. Omitted fields are unmanaged.\n"), b...), nil
}
