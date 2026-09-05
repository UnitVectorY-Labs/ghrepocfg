package github

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

func (c *Client) UpdateRepository(ctx context.Context, owner, repo string, body map[string]any) error {
	_, err := c.request(ctx, http.MethodPatch, repoPath(owner, repo, ""), body, nil)
	return err
}

func (c *Client) ReplaceTopics(ctx context.Context, owner, repo string, topics []string) error {
	_, err := c.request(ctx, http.MethodPut, repoPath(owner, repo, "/topics"), map[string]any{"names": topics}, nil)
	return err
}

func (c *Client) SetCustomProperty(ctx context.Context, owner, repo, name string, value config.CustomPropertyValue) error {
	body := map[string]any{"properties": []map[string]any{{"property_name": name, "value": value.Value}}}
	_, err := c.request(ctx, http.MethodPatch, repoPath(owner, repo, "/properties/values"), body, nil)
	return err
}

func (c *Client) UpdateSecurityAnalysis(ctx context.Context, owner, repo string, body map[string]any) error {
	_, err := c.request(ctx, http.MethodPatch, repoPath(owner, repo, ""), map[string]any{"security_and_analysis": body}, nil)
	return err
}

func (c *Client) SetVulnerabilityAlerts(ctx context.Context, owner, repo string, enabled bool) error {
	method := http.MethodDelete
	if enabled {
		method = http.MethodPut
	}
	_, err := c.request(ctx, method, repoPath(owner, repo, "/vulnerability-alerts"), nil, nil)
	return err
}
func (c *Client) SetAutomatedSecurityFixes(ctx context.Context, owner, repo string, enabled bool) error {
	method := http.MethodDelete
	if enabled {
		method = http.MethodPut
	}
	_, err := c.request(ctx, method, repoPath(owner, repo, "/automated-security-fixes"), nil, nil)
	return err
}

func (c *Client) SetActionsPermissions(ctx context.Context, owner, repo string, body map[string]any) error {
	_, err := c.request(ctx, http.MethodPut, repoPath(owner, repo, "/actions/permissions"), body, nil)
	return err
}
func (c *Client) SetSelectedActions(ctx context.Context, owner, repo string, body *config.SelectedActions) error {
	_, err := c.request(ctx, http.MethodPut, repoPath(owner, repo, "/actions/permissions/selected-actions"), body, nil)
	return err
}
func (c *Client) SetWorkflowPermissions(ctx context.Context, owner, repo string, body map[string]any) error {
	_, err := c.request(ctx, http.MethodPut, repoPath(owner, repo, "/actions/permissions/workflow"), body, nil)
	return err
}

func accessPermission(p string) string { return strings.TrimPrefix(p, "custom:") }
func (c *Client) SetCollaborator(ctx context.Context, owner, repo, user, permission string) error {
	_, err := c.request(ctx, http.MethodPut, repoPath(owner, repo, "/collaborators/"+url.PathEscape(user)), map[string]string{"permission": accessPermission(permission)}, nil)
	return err
}
func (c *Client) RemoveCollaborator(ctx context.Context, owner, repo, user string, invitationID *int64) error {
	path := repoPath(owner, repo, "/collaborators/"+url.PathEscape(user))
	if invitationID != nil {
		path = repoPath(owner, repo, "/invitations/"+intPath(*invitationID))
	}
	_, err := c.request(ctx, http.MethodDelete, path, nil, nil)
	return err
}

func (c *Client) SetTeam(ctx context.Context, owner, repo, slug, permission string) error {
	path := "/orgs/" + url.PathEscape(owner) + "/teams/" + url.PathEscape(slug) + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
	_, err := c.request(ctx, http.MethodPut, path, map[string]string{"permission": accessPermission(permission)}, nil)
	return err
}
func (c *Client) RemoveTeam(ctx context.Context, owner, repo, slug string) error {
	path := "/orgs/" + url.PathEscape(owner) + "/teams/" + url.PathEscape(slug) + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
	_, err := c.request(ctx, http.MethodDelete, path, nil, nil)
	return err
}

func rulesetBody(name string, v config.Ruleset) map[string]any {
	target := v.Target
	if target == "" {
		target = "branch"
	}
	bypass := v.BypassActors
	if bypass == nil {
		bypass = []config.BypassActor{}
	}
	rules := v.Rules
	if rules == nil {
		rules = []config.Rule{}
	}
	body := map[string]any{"name": name, "target": target, "enforcement": v.Enforcement, "bypass_actors": bypass, "rules": rules}
	if v.Conditions.RefName != nil {
		body["conditions"] = v.Conditions
	}
	return body
}
func (c *Client) CreateRuleset(ctx context.Context, owner, repo, name string, v config.Ruleset) error {
	_, err := c.request(ctx, http.MethodPost, repoPath(owner, repo, "/rulesets"), rulesetBody(name, v), nil)
	return err
}
func (c *Client) UpdateRuleset(ctx context.Context, owner, repo, name string, id int64, v config.Ruleset) error {
	_, err := c.request(ctx, http.MethodPut, repoPath(owner, repo, "/rulesets/"+intPath(id)), rulesetBody(name, v), nil)
	return err
}
func (c *Client) RemoveRuleset(ctx context.Context, owner, repo string, id int64) error {
	_, err := c.request(ctx, http.MethodDelete, repoPath(owner, repo, "/rulesets/"+intPath(id)), nil, nil)
	return err
}
