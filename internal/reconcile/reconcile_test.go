package reconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

type fakeExec struct {
	calls []string
	fail  map[string]bool
}

func (f *fakeExec) call(s string) error {
	f.calls = append(f.calls, s)
	if f.fail[s] {
		return errors.New("boom")
	}
	return nil
}
func (f *fakeExec) UpdateRepository(context.Context, string, string, map[string]any) error {
	return f.call("repository")
}
func (f *fakeExec) ReplaceTopics(context.Context, string, string, []string) error {
	return f.call("topics")
}
func (f *fakeExec) SetCustomProperty(_ context.Context, _, _, name string, _ config.CustomPropertyValue) error {
	return f.call("custom-property:" + name)
}
func (f *fakeExec) UpdateSecurityAnalysis(context.Context, string, string, map[string]any) error {
	return f.call("security-analysis")
}
func (f *fakeExec) SetVulnerabilityAlerts(context.Context, string, string, bool) error {
	return f.call("alerts")
}
func (f *fakeExec) SetAutomatedSecurityFixes(context.Context, string, string, bool) error {
	return f.call("fixes")
}
func (f *fakeExec) SetActionsPermissions(context.Context, string, string, map[string]any) error {
	return f.call("actions")
}
func (f *fakeExec) SetSelectedActions(context.Context, string, string, *config.SelectedActions) error {
	return f.call("selected")
}
func (f *fakeExec) SetWorkflowPermissions(context.Context, string, string, map[string]any) error {
	return f.call("workflow")
}
func (f *fakeExec) SetCollaborator(context.Context, string, string, string, string) error {
	return f.call("set-collaborator")
}
func (f *fakeExec) RemoveCollaborator(context.Context, string, string, string, *int64) error {
	return f.call("remove-collaborator")
}
func (f *fakeExec) SetTeam(context.Context, string, string, string, string) error {
	return f.call("set-team")
}
func (f *fakeExec) RemoveTeam(context.Context, string, string, string) error {
	return f.call("remove-team")
}
func (f *fakeExec) CreateRuleset(context.Context, string, string, string, config.Ruleset) error {
	return f.call("create-ruleset")
}
func (f *fakeExec) UpdateRuleset(context.Context, string, string, string, int64, config.Ruleset) error {
	return f.call("update-ruleset")
}
func (f *fakeExec) RemoveRuleset(context.Context, string, string, int64) error {
	return f.call("remove-ruleset")
}
func ptr[T any](v T) *T { return &v }

func TestBuildManagedScalarsAndAuthoritativeCollections(t *testing.T) {
	wantCollabs := map[string]config.Access{"Alice": {Permission: "push"}}
	wantTeams := map[string]config.Access{}
	wantRules := map[string]config.Ruleset{"main": {Enforcement: "active", Rules: []config.Rule{}}}
	desired := &config.Config{Repository: &config.RepositorySettings{HasWiki: ptr(false)}, Collaborators: &wantCollabs, Teams: &wantTeams, Rulesets: &wantRules}
	current := &github.State{Repository: &config.RepositorySettings{HasWiki: ptr(true), HasIssues: ptr(true)}, Collaborators: map[string]github.Collaborator{"alice": {Permission: "pull"}, "bob": {Permission: "pull"}}, Teams: map[string]github.Team{"old": {Permission: "push"}}, Rulesets: map[string]github.Ruleset{"main": {ID: 1, Value: config.Ruleset{Target: "branch", Enforcement: "active", Rules: nil}}, "old": {ID: 2, Value: config.Ruleset{Target: "tag", Enforcement: "active"}}}}
	p := Build("acme", "repo", desired, current, &fakeExec{}, false)
	if !p.Drift {
		t.Fatal("expected drift")
	}
	got := map[string]Operation{}
	for _, c := range p.Changes {
		got[c.Path] = c.Operation
	}
	wants := map[string]Operation{"repository.has_wiki": Modify, "collaborators.Alice.permission": Modify, "collaborators.bob": Remove, "teams.old": Remove, "rulesets.old": Remove}
	if len(got) != len(wants) {
		t.Fatalf("changes = %#v", got)
	}
	for path, op := range wants {
		if got[path] != op {
			t.Errorf("%s = %q want %q", path, got[path], op)
		}
	}
	if _, ok := got["repository.has_issues"]; ok {
		t.Fatal("unmanaged scalar was changed")
	}
	if _, ok := got["rulesets.main"]; ok {
		t.Fatal("nil and empty rules must compare equally")
	}
}

func TestExecuteContinuesAfterFailure(t *testing.T) {
	want := map[string]config.Access{"alice": {Permission: "push"}, "bob": {Permission: "push"}}
	p := Build("o", "r", &config.Config{Collaborators: &want}, &github.State{Collaborators: map[string]github.Collaborator{}}, &fakeExec{}, false)
	f := &fakeExec{fail: map[string]bool{"set-collaborator": true}}
	p = Build("o", "r", &config.Config{Collaborators: &want}, &github.State{Collaborators: map[string]github.Collaborator{}}, f, false)
	succeeded, failed := p.Execute(context.Background())
	if len(f.calls) != 2 || len(failed) != 2 || len(succeeded) != 0 {
		t.Fatalf("calls=%v succeeded=%v failed=%v", f.calls, succeeded, failed)
	}
}

func TestCustomPropertiesAreAuthoritativeAndFailuresAreIsolated(t *testing.T) {
	want := map[string]config.CustomPropertyValue{
		"editable":    {Value: "new"},
		"restricted":  {Value: "blocked"},
		"already_off": {},
		"unchanged":   {Value: []string{"two", "one"}},
	}
	got := map[string]config.CustomPropertyValue{
		"editable":      {Value: "old"},
		"restricted":    {Value: "old"},
		"removed":       {Value: []string{"legacy"}},
		"already_unset": {},
		"unchanged":     {Value: []string{"one", "two"}},
	}
	f := &fakeExec{fail: map[string]bool{"custom-property:restricted": true}}
	p := Build("o", "r", &config.Config{CustomProperties: &want}, &github.State{CustomProperties: got}, f, false)
	if len(p.Changes) != 3 {
		t.Fatalf("changes = %#v", p.Changes)
	}
	wantPaths := []string{"custom_properties.editable", "custom_properties.removed", "custom_properties.restricted"}
	for i, path := range wantPaths {
		if p.Changes[i].Path != path {
			t.Fatalf("change[%d] = %q, want %q", i, p.Changes[i].Path, path)
		}
	}
	succeeded, failed := p.Execute(context.Background())
	if len(succeeded) != 2 || len(failed) != 1 || failed[0].Path != "custom_properties.restricted" || len(f.calls) != 3 {
		t.Fatalf("calls=%v succeeded=%v failed=%v", f.calls, succeeded, failed)
	}
}

func TestIdempotentPlan(t *testing.T) {
	d := &config.Config{Repository: &config.RepositorySettings{HasWiki: ptr(true)}}
	s := &github.State{Repository: &config.RepositorySettings{HasWiki: ptr(true), HasIssues: ptr(true)}}
	p := Build("o", "r", d, s, &fakeExec{}, true)
	if p.Drift || len(p.Changes) != 0 {
		t.Fatalf("unexpected drift: %#v", p.Changes)
	}
	found := false
	for _, path := range p.Unmanaged {
		if path == "repository.has_issues" {
			found = true
		}
	}
	if !found {
		t.Fatalf("verbose unmanaged = %v", p.Unmanaged)
	}
}

func TestRulesetUpdateFalseMatchesGitHubOmittedParameters(t *testing.T) {
	want := map[string]config.Ruleset{"tag": {
		Target: "tag", Enforcement: "active",
		Rules: []config.Rule{{Type: "update", Parameters: &config.RuleParameters{UpdateAllowsFetchAndMerge: ptr(false)}}},
	}}
	got := map[string]github.Ruleset{"tag": {
		ID: 1, Value: config.Ruleset{Target: "tag", Enforcement: "active", Rules: []config.Rule{{Type: "update"}}},
	}}
	p := Build("o", "r", &config.Config{Rulesets: &want}, &github.State{Rulesets: got}, &fakeExec{}, false)
	if p.Drift || len(p.Changes) != 0 {
		t.Fatalf("GitHub-omitted false update parameter produced drift: %#v", p.Changes)
	}
}

func TestRulesetUpdateTrueDoesNotMatchGitHubOmittedParameters(t *testing.T) {
	want := map[string]config.Ruleset{"branch": {
		Enforcement: "active",
		Rules:       []config.Rule{{Type: "update", Parameters: &config.RuleParameters{UpdateAllowsFetchAndMerge: ptr(true)}}},
	}}
	got := map[string]github.Ruleset{"branch": {
		ID: 1, Value: config.Ruleset{Target: "branch", Enforcement: "active", Rules: []config.Rule{{Type: "update"}}},
	}}
	p := Build("o", "r", &config.Config{Rulesets: &want}, &github.State{Rulesets: got}, &fakeExec{}, false)
	if !p.Drift || len(p.Changes) != 1 || p.Changes[0].Path != "rulesets.branch" {
		t.Fatalf("explicit true update parameter must remain drift: %#v", p.Changes)
	}
}
