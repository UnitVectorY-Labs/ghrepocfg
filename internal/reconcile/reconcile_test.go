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
