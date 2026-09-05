package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

type State struct {
	Repository       *config.RepositorySettings
	CustomProperties map[string]config.CustomPropertyValue
	Security         *config.SecuritySettings
	Actions          *config.ActionsSettings
	Collaborators    map[string]Collaborator
	Teams            map[string]Team
	Rulesets         map[string]Ruleset
	Warnings         []string
	UnknownFields    []string
}

type Collaborator struct {
	Permission   string
	InvitationID *int64
}
type Team struct{ Permission string }
type Ruleset struct {
	ID    int64
	Value config.Ruleset
}

type ReadScope struct {
	Repository, CustomProperties, Security, Actions, Collaborators, Teams, Rulesets bool
	SelectedActions                                                                 bool
	Verbose                                                                         bool
}

func (c *Client) Read(ctx context.Context, owner, repo string, scope ReadScope) (*State, error) {
	s := &State{}
	// Repository metadata is always read: it proves access and supports warnings.
	var raw map[string]json.RawMessage
	if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, ""), nil, &raw); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(raw)
	var repositoryState config.RepositorySettings
	if err := json.Unmarshal(b, &repositoryState); err != nil {
		return nil, err
	}
	// GitHub represents unset textual fields as null; ghrepocfg's portable
	// desired value for them is the empty string.
	if _, ok := raw["description"]; ok && repositoryState.Description == nil {
		empty := ""
		repositoryState.Description = &empty
	}
	if _, ok := raw["homepage"]; ok && repositoryState.Homepage == nil {
		empty := ""
		repositoryState.Homepage = &empty
	}
	s.Repository = &repositoryState
	if scope.Verbose {
		s.UnknownFields = unknownRepositoryFields(raw)
	}
	var permissions struct {
		Admin bool `json:"admin"`
	}
	if value, ok := raw["permissions"]; ok {
		_ = json.Unmarshal(value, &permissions)
	}
	if (scope.Security || scope.Collaborators || scope.Teams || scope.Rulesets) && !permissions.Admin {
		return nil, fmt.Errorf("repository admin access is required to safely read managed security, access, and ruleset state")
	}
	if scope.Security {
		v, err := c.readSecurity(ctx, owner, repo, raw["security_and_analysis"])
		if err != nil {
			return nil, fmt.Errorf("read security settings: %w", err)
		}
		s.Security = v
	}
	if scope.CustomProperties {
		v, err := c.readCustomProperties(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("read custom properties: %w", err)
		}
		s.CustomProperties = v
	}
	if scope.Actions {
		v, err := c.readActions(ctx, owner, repo, scope.SelectedActions)
		if err != nil {
			return nil, fmt.Errorf("read Actions settings: %w", err)
		}
		s.Actions = v
	}
	if scope.Collaborators {
		v, err := c.readCollaborators(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("read collaborators: %w", err)
		}
		s.Collaborators = v
	}
	if scope.Teams {
		v, err := c.readTeams(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("read team access: %w", err)
		}
		s.Teams = v
	}
	if scope.Rulesets {
		v, err := c.readRulesets(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("read rulesets: %w", err)
		}
		s.Rulesets = v
	}
	s.Warnings = append(s.Warnings, c.legacyWarnings(ctx, owner, repo)...)
	return s, nil
}

func (c *Client) readCustomProperties(ctx context.Context, owner, repo string) (map[string]config.CustomPropertyValue, error) {
	var values []struct {
		PropertyName string                     `json:"property_name"`
		Value        config.CustomPropertyValue `json:"value"`
	}
	if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/properties/values"), nil, &values); err != nil {
		return nil, err
	}
	result := make(map[string]config.CustomPropertyValue, len(values))
	for _, property := range values {
		result[property.PropertyName] = property.Value
	}
	return result, nil
}

func unknownRepositoryFields(raw map[string]json.RawMessage) []string {
	// This includes both managed fields and deliberately recognized read-only or
	// high-risk response fields. Verbose output is reserved for genuinely new
	// REST response keys, rather than listing GitHub's normal metadata.
	names := []string{"id", "node_id", "name", "full_name", "private", "owner", "html_url", "description", "fork", "url", "forks_url", "keys_url", "collaborators_url", "teams_url", "hooks_url", "issue_events_url", "events_url", "assignees_url", "branches_url", "tags_url", "blobs_url", "git_tags_url", "git_refs_url", "trees_url", "statuses_url", "languages_url", "stargazers_url", "contributors_url", "subscribers_url", "subscription_url", "commits_url", "git_commits_url", "comments_url", "issue_comment_url", "contents_url", "compare_url", "merges_url", "archive_url", "downloads_url", "issues_url", "pulls_url", "milestones_url", "notifications_url", "labels_url", "releases_url", "deployments_url", "created_at", "updated_at", "pushed_at", "git_url", "ssh_url", "clone_url", "svn_url", "homepage", "size", "stargazers_count", "watchers_count", "language", "has_issues", "has_projects", "has_downloads", "has_wiki", "has_pages", "has_discussions", "has_pull_requests", "pull_request_creation_policy", "forks_count", "mirror_url", "archived", "disabled", "open_issues_count", "license", "allow_forking", "is_template", "web_commit_signoff_required", "topics", "visibility", "forks", "open_issues", "watchers", "default_branch", "temp_clone_token", "network_count", "subscribers_count", "organization", "permissions", "security_and_analysis", "allow_squash_merge", "allow_merge_commit", "allow_rebase_merge", "allow_auto_merge", "delete_branch_on_merge", "allow_update_branch", "use_squash_pr_title_as_default", "squash_merge_commit_title", "squash_merge_commit_message", "merge_commit_title", "merge_commit_message", "custom_properties"}
	known := make(map[string]bool, len(names))
	for _, name := range names {
		known[name] = true
	}
	var fields []string
	for k := range raw {
		if !known[k] {
			fields = append(fields, k)
		}
	}
	sort.Strings(fields)
	return fields
}

func (c *Client) readSecurity(ctx context.Context, owner, repo string, analysisRaw json.RawMessage) (*config.SecuritySettings, error) {
	alerts, err := c.booleanEndpoint(ctx, repoPath(owner, repo, "/vulnerability-alerts"))
	if err != nil {
		return nil, err
	}
	fixes, err := c.booleanEndpoint(ctx, repoPath(owner, repo, "/automated-security-fixes"))
	if err != nil {
		return nil, err
	}
	v := &config.SecuritySettings{VulnerabilityAlerts: &alerts, AutomatedSecurityFixes: &fixes}
	if len(analysisRaw) > 0 && string(analysisRaw) != "null" {
		_ = json.Unmarshal(analysisRaw, v)
	}
	return v, nil
}

func (c *Client) booleanEndpoint(ctx context.Context, path string) (bool, error) {
	_, err := c.request(ctx, http.MethodGet, path, nil, nil)
	if err == nil {
		return true, nil
	}
	if IsStatus(err, http.StatusNotFound) {
		return false, nil
	}
	return false, err
}

func (c *Client) readActions(ctx context.Context, owner, repo string, readSelected bool) (*config.ActionsSettings, error) {
	var permissions struct {
		Enabled        bool   `json:"enabled"`
		AllowedActions string `json:"allowed_actions"`
	}
	if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/actions/permissions"), nil, &permissions); err != nil {
		return nil, err
	}
	v := &config.ActionsSettings{Enabled: &permissions.Enabled, AllowedActions: &permissions.AllowedActions}
	if permissions.AllowedActions == "selected" || readSelected {
		var selected config.SelectedActions
		if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/actions/permissions/selected-actions"), nil, &selected); err != nil {
			return nil, err
		}
		v.SelectedActions = &selected
	}
	var workflow struct {
		Default string `json:"default_workflow_permissions"`
		Approve bool   `json:"can_approve_pull_request_reviews"`
	}
	if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/actions/permissions/workflow"), nil, &workflow); err != nil {
		return nil, err
	}
	v.DefaultWorkflowPermissions, v.CanApprovePullRequestReviews = &workflow.Default, &workflow.Approve
	return v, nil
}

func (c *Client) readCollaborators(ctx context.Context, owner, repo string) (map[string]Collaborator, error) {
	var users []struct {
		Login, RoleName string
		Permissions     map[string]bool
	}
	if err := c.paged(ctx, repoPath(owner, repo, "/collaborators?affiliation=direct"), &users); err != nil {
		return nil, err
	}
	result := make(map[string]Collaborator, len(users))
	for _, u := range users {
		result[strings.ToLower(u.Login)] = Collaborator{Permission: permission(u.RoleName, u.Permissions)}
	}
	var invitations []struct {
		ID                      int64
		Permissions, Permission string
		Invitee                 *struct{ Login string }
	}
	if err := c.paged(ctx, repoPath(owner, repo, "/invitations"), &invitations); err != nil {
		return nil, err
	}
	for _, inv := range invitations {
		if inv.Invitee != nil {
			p := inv.Permissions
			if p == "" {
				p = inv.Permission
			}
			id := inv.ID
			result[strings.ToLower(inv.Invitee.Login)] = Collaborator{Permission: p, InvitationID: &id}
		}
	}
	return result, nil
}

func permission(role string, p map[string]bool) string {
	switch role {
	case "read":
		return "pull"
	case "write":
		return "push"
	case "triage", "maintain", "admin", "pull", "push":
		return role
	}
	if role != "" {
		return "custom:" + role
	}
	for _, name := range []string{"admin", "maintain", "push", "triage", "pull"} {
		if p[name] {
			return name
		}
	}
	return "pull"
}

func (c *Client) readTeams(ctx context.Context, owner, repo string) (map[string]Team, error) {
	var teams []struct {
		Slug, Permission string
		Permissions      map[string]bool
	}
	if err := c.paged(ctx, repoPath(owner, repo, "/teams"), &teams); err != nil {
		return nil, err
	}
	result := make(map[string]Team, len(teams))
	for _, t := range teams {
		result[strings.ToLower(t.Slug)] = Team{Permission: permission(t.Permission, t.Permissions)}
	}
	return result, nil
}

func (c *Client) readRulesets(ctx context.Context, owner, repo string) (map[string]Ruleset, error) {
	var summaries []struct {
		ID               int64
		Name, SourceType string
	}
	if err := c.paged(ctx, repoPath(owner, repo, "/rulesets?includes_parents=false"), &summaries); err != nil {
		return nil, err
	}
	result := make(map[string]Ruleset, len(summaries))
	for _, summary := range summaries {
		if summary.SourceType != "" && summary.SourceType != "Repository" {
			continue
		}
		var detail struct {
			ID           int64                `json:"id"`
			Name         string               `json:"name"`
			Target       string               `json:"target"`
			Enforcement  string               `json:"enforcement"`
			BypassActors []config.BypassActor `json:"bypass_actors"`
			Conditions   config.Conditions    `json:"conditions"`
			Rules        []config.Rule        `json:"rules"`
		}
		if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/rulesets/"+intPath(summary.ID)), nil, &detail); err != nil {
			return nil, err
		}
		if _, exists := result[detail.Name]; exists {
			return nil, fmt.Errorf("repository contains duplicate ruleset name %q; names must be unique for portable reconciliation", detail.Name)
		}
		result[detail.Name] = Ruleset{ID: detail.ID, Value: config.Ruleset{Target: detail.Target, Enforcement: detail.Enforcement, BypassActors: detail.BypassActors, Conditions: detail.Conditions, Rules: detail.Rules}}
	}
	return result, nil
}

func (c *Client) legacyWarnings(ctx context.Context, owner, repo string) []string {
	var warnings []string
	var branches []struct {
		Name      string
		Protected bool
	}
	if err := c.paged(ctx, repoPath(owner, repo, "/branches?protected=true"), &branches); err == nil {
		for _, b := range branches {
			if !b.Protected {
				continue
			}
			if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/branches/"+url.PathEscape(b.Name)+"/protection"), nil, nil); err == nil {
				warnings = append(warnings, "legacy branch protection exists for "+b.Name+" and is not managed")
			}
		}
	}
	var tags []json.RawMessage
	if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/tags/protection"), nil, &tags); err == nil && len(tags) > 0 {
		warnings = append(warnings, "legacy tag protection exists and is not managed")
	}
	return warnings
}
