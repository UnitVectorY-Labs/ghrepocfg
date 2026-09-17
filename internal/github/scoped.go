package github

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

func wantedField(scope ReadScope, section, field string) bool {
	if scope.Full || scope.Desired == nil {
		return true
	}
	v := reflect.ValueOf(scope.Desired).Elem()
	for i := 0; i < v.NumField(); i++ {
		if strings.Split(v.Type().Field(i).Tag.Get("json"), ",")[0] != section {
			continue
		}
		if v.Field(i).IsNil() {
			return false
		}
		v = v.Field(i).Elem()
		for j := 0; j < v.NumField(); j++ {
			if strings.Split(v.Type().Field(j).Tag.Get("json"), ",")[0] == field {
				return !v.Field(j).IsNil()
			}
		}
		return false
	}
	return false
}

func (c *Client) readSecurityScoped(ctx context.Context, owner, repo string, raw json.RawMessage, scope ReadScope, s *State) error {
	s.Security = &config.SecuritySettings{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, s.Security); err != nil {
			return err
		}
	}
	v := reflect.ValueOf(s.Security).Elem()
	for i := 0; i < v.NumField(); i++ {
		name := strings.Split(v.Type().Field(i).Tag.Get("json"), ",")[0]
		switch name {
		case "vulnerability_alerts", "automated_security_fixes", "private_vulnerability_reporting", "code_scanning_default_setup":
			continue
		}
		if v.Field(i).IsNil() && wantedField(scope, "security", name) {
			s.Omit([]string{"security", name}, "security setting is not exposed by GitHub; access or feature availability may be restricted")
		}
	}
	if wantedField(scope, "security", "vulnerability_alerts") {
		if err := s.read([]string{"security.vulnerability_alerts"}, func() error {
			_, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/vulnerability-alerts"), nil, nil)
			// A 404 can also conceal insufficient permissions. Do not invent false.
			if err != nil {
				return err
			}
			yes := true
			s.Security.VulnerabilityAlerts = &yes
			return nil
		}); err != nil {
			return err
		}
	}
	if wantedField(scope, "security", "automated_security_fixes") {
		if err := s.read([]string{"security.automated_security_fixes"}, func() error {
			var v struct {
				Enabled *bool `json:"enabled"`
			}
			if _, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/automated-security-fixes"), nil, &v); err != nil {
				return err
			}
			if v.Enabled == nil {
				return &unreadableState{reason: "security updates enabled state is not exposed by GitHub"}
			}
			s.Security.AutomatedSecurityFixes = v.Enabled
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) readActionsScoped(ctx context.Context, owner, repo string, scope ReadScope, s *State) error {
	s.Actions = &config.ActionsSettings{}
	groups := []struct {
		suffix string
		fields []string
	}{
		{"", []string{"enabled", "allowed_actions", "sha_pinning_required"}},
		{"/workflow", []string{"default_workflow_permissions", "can_approve_pull_request_reviews"}},
	}
	for _, g := range groups {
		wanted := false
		paths := []string{}
		for _, f := range g.fields {
			if wantedField(scope, "actions", f) {
				wanted = true
				paths = append(paths, "actions."+f)
			}
		}
		// Selected-action writes depend on the permissions policy.
		if g.suffix == "" && scope.SelectedActions {
			wanted = true
			paths = append(paths, "actions.selected_actions")
		}
		if !wanted {
			continue
		}
		if err := s.read(paths, func() error {
			if err := requestInto(ctx, c, owner, repo, "/actions/permissions"+g.suffix, s.Actions); err != nil {
				return err
			}
			if g.suffix == "" && s.Actions.Enabled == nil {
				return &unreadableState{reason: "required Actions enabled state is not exposed by GitHub"}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	selected := scope.SelectedActions || (s.Actions.AllowedActions != nil && *s.Actions.AllowedActions == "selected" && (scope.Full || scope.Desired == nil))
	if selected {
		for _, u := range s.Unavailable {
			if u.Path == "actions.enabled" || u.Path == "actions.allowed_actions" || u.Path == "actions.selected_actions" {
				s.Omit([]string{"actions", "selected_actions"}, "required Actions permissions policy could not be read")
				return nil
			}
		}
		if err := s.read([]string{"actions.selected_actions"}, func() error {
			var v config.SelectedActions
			if err := requestInto(ctx, c, owner, repo, "/actions/permissions/selected-actions", &v); err != nil {
				return err
			}
			s.Actions.SelectedActions = &v
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
func requestInto(ctx context.Context, c *Client, owner, repo, path string, out any) error {
	_, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, path), nil, out)
	return err
}

// Requested pointer fields absent from successful responses remain unknown.
func (s *State) markMissing(desired *config.Config) {
	if desired == nil {
		return
	}
	current := s.Additional
	current.Repository, current.Security, current.Actions = s.Repository, s.Security, s.Actions
	var walk func(reflect.Value, reflect.Value, []string)
	walk = func(want, got reflect.Value, path []string) {
		if want.Kind() == reflect.Pointer {
			if want.IsNil() {
				return
			}
			if !got.IsValid() || got.IsNil() {
				s.Omit(path, "requested value is not exposed by GitHub")
				return
			}
			want, got = want.Elem(), got.Elem()
		}
		if want.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < want.NumField(); i++ {
			name := strings.Split(want.Type().Field(i).Tag.Get("json"), ",")[0]
			// Authoritative collections are handled by their readers, including pagination.
			field := want.Field(i)
			if field.Kind() != reflect.Pointer || field.Type().Elem().Kind() == reflect.Map {
				continue
			}
			walk(field, got.Field(i), append(append([]string{}, path...), name))
		}
	}
	walk(reflect.ValueOf(desired).Elem(), reflect.ValueOf(current), nil)
}
