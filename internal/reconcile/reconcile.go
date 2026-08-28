// Package reconcile turns a strictly validated desired config and a fully read
// GitHub state into a structured, executable plan.
package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

type Operation string

const (
	Add    Operation = "add"
	Remove Operation = "remove"
	Modify Operation = "modify"
)

type Change struct {
	Operation Operation `json:"operation"`
	Path      string    `json:"path"`
	Before    any       `json:"before,omitempty"`
	After     any       `json:"after,omitempty"`
	apply     func(context.Context) error
}

type Plan struct {
	Repository string   `json:"repository"`
	Drift      bool     `json:"drift"`
	Changes    []Change `json:"changes"`
	Warnings   []string `json:"warnings,omitempty"`
	Unmanaged  []string `json:"unmanaged,omitempty"`
}

type Executor interface {
	UpdateRepository(context.Context, string, string, map[string]any) error
	ReplaceTopics(context.Context, string, string, []string) error
	UpdateSecurityAnalysis(context.Context, string, string, map[string]any) error
	SetVulnerabilityAlerts(context.Context, string, string, bool) error
	SetAutomatedSecurityFixes(context.Context, string, string, bool) error
	SetActionsPermissions(context.Context, string, string, map[string]any) error
	SetSelectedActions(context.Context, string, string, *config.SelectedActions) error
	SetWorkflowPermissions(context.Context, string, string, map[string]any) error
	SetCollaborator(context.Context, string, string, string, string) error
	RemoveCollaborator(context.Context, string, string, string, *int64) error
	SetTeam(context.Context, string, string, string, string) error
	RemoveTeam(context.Context, string, string, string) error
	CreateRuleset(context.Context, string, string, string, config.Ruleset) error
	UpdateRuleset(context.Context, string, string, string, int64, config.Ruleset) error
	RemoveRuleset(context.Context, string, string, int64) error
}

func Build(owner, repo string, desired *config.Config, current *github.State, exec Executor, verbose bool) *Plan {
	p := &Plan{Repository: owner + "/" + repo, Changes: make([]Change, 0), Warnings: append([]string(nil), current.Warnings...)}
	if desired.Repository != nil {
		repositoryChanges(p, owner, repo, desired.Repository, current.Repository, exec)
	}
	if verbose {
		p.Unmanaged = append(p.Unmanaged, unmanagedRepository(desired.Repository, current.Repository)...)
	}
	if desired.Security != nil {
		securityChanges(p, owner, repo, desired.Security, current.Security, exec)
	} else if verbose {
		p.Unmanaged = append(p.Unmanaged, "security")
	}
	if desired.Actions != nil {
		actionsChanges(p, owner, repo, desired.Actions, current.Actions, exec)
	} else if verbose {
		p.Unmanaged = append(p.Unmanaged, "actions")
	}
	if desired.Collaborators != nil {
		collaboratorChanges(p, owner, repo, *desired.Collaborators, current.Collaborators, exec)
	} else if verbose {
		p.Unmanaged = append(p.Unmanaged, "collaborators")
	}
	if desired.Teams != nil {
		teamChanges(p, owner, repo, *desired.Teams, current.Teams, exec)
	} else if verbose {
		p.Unmanaged = append(p.Unmanaged, "teams")
	}
	if desired.Rulesets != nil {
		rulesetChanges(p, owner, repo, *desired.Rulesets, current.Rulesets, exec)
	} else if verbose {
		p.Unmanaged = append(p.Unmanaged, "rulesets")
	}
	p.Drift = len(p.Changes) > 0
	return p
}

func add(p *Plan, c Change) { p.Changes = append(p.Changes, c) }

func repositoryChanges(p *Plan, owner, repo string, want, got *config.RepositorySettings, e Executor) {
	wv, gv := reflect.ValueOf(want).Elem(), reflect.ValueOf(got).Elem()
	typ := wv.Type()
	for i := 0; i < wv.NumField(); i++ {
		if wv.Field(i).IsNil() {
			continue
		}
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		before := gv.Field(i).Interface()
		after := wv.Field(i).Interface()
		if valuesEqual(before, after) {
			continue
		}
		b := indirect(before)
		a := indirect(after)
		if tag == "topics" {
			source := a.([]string)
			topics := make([]string, len(source))
			copy(topics, source)
			add(p, Change{Operation: Modify, Path: "repository.topics", Before: b, After: a, apply: func(ctx context.Context) error { return e.ReplaceTopics(ctx, owner, repo, topics) }})
			continue
		}
		body := map[string]any{tag: a}
		add(p, Change{Operation: Modify, Path: "repository." + tag, Before: b, After: a, apply: func(ctx context.Context) error { return e.UpdateRepository(ctx, owner, repo, body) }})
	}
}

func securityChanges(p *Plan, owner, repo string, want, got *config.SecuritySettings, e Executor) {
	if want.VulnerabilityAlerts != nil && !valuesEqual(want.VulnerabilityAlerts, got.VulnerabilityAlerts) {
		v := *want.VulnerabilityAlerts
		add(p, Change{Modify, "security.vulnerability_alerts", indirect(got.VulnerabilityAlerts), v, func(ctx context.Context) error { return e.SetVulnerabilityAlerts(ctx, owner, repo, v) }})
	}
	if want.AutomatedSecurityFixes != nil && !valuesEqual(want.AutomatedSecurityFixes, got.AutomatedSecurityFixes) {
		v := *want.AutomatedSecurityFixes
		add(p, Change{Modify, "security.automated_security_fixes", indirect(got.AutomatedSecurityFixes), v, func(ctx context.Context) error { return e.SetAutomatedSecurityFixes(ctx, owner, repo, v) }})
	}
	wv, gv := reflect.ValueOf(want).Elem(), reflect.ValueOf(got).Elem()
	typ := wv.Type()
	body := map[string]any{}
	before := map[string]any{}
	for i := 0; i < wv.NumField(); i++ {
		if wv.Field(i).IsNil() {
			continue
		}
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if tag == "vulnerability_alerts" || tag == "automated_security_fixes" {
			continue
		}
		if valuesEqual(wv.Field(i).Interface(), gv.Field(i).Interface()) {
			continue
		}
		body[tag] = indirect(wv.Field(i).Interface())
		before[tag] = indirect(gv.Field(i).Interface())
	}
	if len(body) > 0 {
		payload := body
		add(p, Change{Operation: Modify, Path: "security.security_and_analysis", Before: before, After: payload, apply: func(ctx context.Context) error { return e.UpdateSecurityAnalysis(ctx, owner, repo, payload) }})
	}
}

func actionsChanges(p *Plan, owner, repo string, want, got *config.ActionsSettings, e Executor) {
	permissions := map[string]any{}
	var beforePerm = map[string]any{}
	if want.Enabled != nil && !valuesEqual(want.Enabled, got.Enabled) {
		permissions["enabled"] = *want.Enabled
		beforePerm["enabled"] = indirect(got.Enabled)
	}
	if want.AllowedActions != nil && !valuesEqual(want.AllowedActions, got.AllowedActions) {
		permissions["allowed_actions"] = *want.AllowedActions
		beforePerm["allowed_actions"] = indirect(got.AllowedActions)
	}
	if len(permissions) > 0 {
		body := permissions
		add(p, Change{Modify, "actions.permissions", beforePerm, body, func(ctx context.Context) error { return e.SetActionsPermissions(ctx, owner, repo, body) }})
	}
	if want.SelectedActions != nil && !valuesEqual(want.SelectedActions, got.SelectedActions) {
		v := want.SelectedActions
		add(p, Change{Modify, "actions.selected_actions", got.SelectedActions, v, func(ctx context.Context) error { return e.SetSelectedActions(ctx, owner, repo, v) }})
	}
	workflow := map[string]any{}
	beforeWorkflow := map[string]any{}
	if want.DefaultWorkflowPermissions != nil && !valuesEqual(want.DefaultWorkflowPermissions, got.DefaultWorkflowPermissions) {
		workflow["default_workflow_permissions"] = *want.DefaultWorkflowPermissions
		beforeWorkflow["default_workflow_permissions"] = indirect(got.DefaultWorkflowPermissions)
	}
	if want.CanApprovePullRequestReviews != nil && !valuesEqual(want.CanApprovePullRequestReviews, got.CanApprovePullRequestReviews) {
		workflow["can_approve_pull_request_reviews"] = *want.CanApprovePullRequestReviews
		beforeWorkflow["can_approve_pull_request_reviews"] = indirect(got.CanApprovePullRequestReviews)
	}
	if len(workflow) > 0 {
		body := workflow
		add(p, Change{Modify, "actions.workflow_permissions", beforeWorkflow, body, func(ctx context.Context) error { return e.SetWorkflowPermissions(ctx, owner, repo, body) }})
	}
}

func normalizeAccess(in map[string]config.Access) map[string]struct {
	Name   string
	Access config.Access
} {
	out := make(map[string]struct {
		Name   string
		Access config.Access
	}, len(in))
	for n, a := range in {
		out[strings.ToLower(n)] = struct {
			Name   string
			Access config.Access
		}{n, a}
	}
	return out
}
func collaboratorChanges(p *Plan, owner, repo string, want map[string]config.Access, got map[string]github.Collaborator, e Executor) {
	w := normalizeAccess(want)
	keys := make([]string, 0, len(w)+len(got))
	seen := map[string]bool{}
	for k := range w {
		seen[k] = true
		keys = append(keys, k)
	}
	for k := range got {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		d, dok := w[key]
		c, cok := got[key]
		switch {
		case dok && !cok:
			name, perm := d.Name, d.Access.Permission
			add(p, Change{Add, "collaborators." + name, nil, perm, func(ctx context.Context) error { return e.SetCollaborator(ctx, owner, repo, name, perm) }})
		case !dok && cok:
			name := key
			cur := c
			add(p, Change{Remove, "collaborators." + name, cur.Permission, nil, func(ctx context.Context) error { return e.RemoveCollaborator(ctx, owner, repo, name, cur.InvitationID) }})
		case dok && cok && d.Access.Permission != c.Permission:
			name, perm := d.Name, d.Access.Permission
			add(p, Change{Modify, "collaborators." + name + ".permission", c.Permission, perm, func(ctx context.Context) error { return e.SetCollaborator(ctx, owner, repo, name, perm) }})
		}
	}
}
func teamChanges(p *Plan, owner, repo string, want map[string]config.Access, got map[string]github.Team, e Executor) {
	w := normalizeAccess(want)
	keys := make([]string, 0, len(w)+len(got))
	seen := map[string]bool{}
	for k := range w {
		seen[k] = true
		keys = append(keys, k)
	}
	for k := range got {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		d, dok := w[key]
		c, cok := got[key]
		switch {
		case dok && !cok:
			name, perm := d.Name, d.Access.Permission
			add(p, Change{Add, "teams." + name, nil, perm, func(ctx context.Context) error { return e.SetTeam(ctx, owner, repo, name, perm) }})
		case !dok && cok:
			name := key
			add(p, Change{Remove, "teams." + name, c.Permission, nil, func(ctx context.Context) error { return e.RemoveTeam(ctx, owner, repo, name) }})
		case dok && cok && d.Access.Permission != c.Permission:
			name, perm := d.Name, d.Access.Permission
			add(p, Change{Modify, "teams." + name + ".permission", c.Permission, perm, func(ctx context.Context) error { return e.SetTeam(ctx, owner, repo, name, perm) }})
		}
	}
}
func rulesetChanges(p *Plan, owner, repo string, want map[string]config.Ruleset, got map[string]github.Ruleset, e Executor) {
	keys := make([]string, 0, len(want)+len(got))
	seen := map[string]bool{}
	for k := range want {
		seen[k] = true
		keys = append(keys, k)
	}
	for k := range got {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, name := range keys {
		d, dok := want[name]
		c, cok := got[name]
		if d.Target == "" {
			d.Target = "branch"
		}
		for i := range d.BypassActors {
			if d.BypassActors[i].BypassMode == "" {
				d.BypassActors[i].BypassMode = "always"
			}
		}
		switch {
		case dok && !cok:
			v := d
			add(p, Change{Add, "rulesets." + name, nil, v, func(ctx context.Context) error { return e.CreateRuleset(ctx, owner, repo, name, v) }})
		case !dok && cok:
			id := c.ID
			add(p, Change{Remove, "rulesets." + name, c.Value, nil, func(ctx context.Context) error { return e.RemoveRuleset(ctx, owner, repo, id) }})
		case dok && cok && !rulesetsEqual(d, c.Value):
			id, v := c.ID, d
			add(p, Change{Modify, "rulesets." + name, c.Value, v, func(ctx context.Context) error { return e.UpdateRuleset(ctx, owner, repo, name, id, v) }})
		}
	}
}

func rulesetsEqual(a, b config.Ruleset) bool {
	a = normalizeRulesetDefaults(a)
	b = normalizeRulesetDefaults(b)
	return valuesEqual(a, b)
}

func normalizeRulesetDefaults(v config.Ruleset) config.Ruleset {
	v.Rules = append([]config.Rule(nil), v.Rules...)
	for i := range v.Rules {
		rule := &v.Rules[i]
		if rule.Type == "update" && (rule.Parameters == nil || rule.Parameters.UpdateAllowsFetchAndMerge == nil || !*rule.Parameters.UpdateAllowsFetchAndMerge) {
			rule.Parameters = nil
		}
	}
	return v
}

func unmanagedRepository(desired, current *config.RepositorySettings) []string {
	var out []string
	cv := reflect.ValueOf(current).Elem()
	ct := cv.Type()
	var dv reflect.Value
	if desired != nil {
		dv = reflect.ValueOf(desired).Elem()
	}
	for i := 0; i < cv.NumField(); i++ {
		if cv.Field(i).IsNil() {
			continue
		}
		if desired != nil && !dv.Field(i).IsNil() {
			continue
		}
		tag := strings.Split(ct.Field(i).Tag.Get("json"), ",")[0]
		out = append(out, "repository."+tag)
	}
	return out
}

func valuesEqual(a, b any) bool {
	ca, _ := canonicalJSON(a)
	cb, _ := canonicalJSON(b)
	return string(ca) == string(cb)
}
func canonicalJSON(v any) ([]byte, error) { return json.Marshal(normalize(reflect.ValueOf(v))) }
func normalize(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return normalize(v.Elem())
	}
	switch v.Kind() {
	case reflect.Struct:
		m := map[string]any{}
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			tag := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
			if tag == "" || tag == "-" {
				continue
			}
			x := normalize(v.Field(i))
			if x != nil {
				m[tag] = x
			}
		}
		return m
	case reflect.Slice:
		if v.IsNil() {
			return []any{}
		}
		a := make([]any, v.Len())
		for i := range a {
			a[i] = normalize(v.Index(i))
		}
		return a
	case reflect.Map:
		if v.IsNil() {
			return map[string]any{}
		}
		m := map[string]any{}
		it := v.MapRange()
		for it.Next() {
			m[fmt.Sprint(it.Key().Interface())] = normalize(it.Value())
		}
		return m
	default:
		return v.Interface()
	}
}
func indirect(v any) any {
	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		return rv.Elem().Interface()
	}
	return v
}

type Failure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

func (p *Plan) Execute(ctx context.Context) (succeeded []string, failed []Failure) {
	for _, c := range p.Changes {
		if err := c.apply(ctx); err != nil {
			failed = append(failed, Failure{c.Path, err.Error()})
		} else {
			succeeded = append(succeeded, c.Path)
		}
	}
	return
}
