package reconcile

import (
	"context"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

type recordingExec struct {
	fakeExec
	bodies      []any
	environment config.Environment
}

func (f *recordingExec) Mutate(_ context.Context, _, _, method, path string, body any) error {
	f.bodies = append(f.bodies, body)
	return f.call(method + " " + path)
}
func (f *recordingExec) SetEnvironment(_ context.Context, _, _, name string, v config.Environment) error {
	f.environment = v
	return f.call("environment:" + name)
}
func TestAdditionalPlansPreserveOmittedFields(t *testing.T) {
	f := &recordingExec{}
	d := &config.Config{Actions: &config.ActionsSettings{SHAPinningRequired: ptr(true), PrivateForkWorkflows: &config.PrivateForkWorkflows{SendWriteTokensToWorkflows: ptr(false)}}}
	s := &github.State{Actions: &config.ActionsSettings{Enabled: ptr(true), SHAPinningRequired: ptr(false), PrivateForkWorkflows: &config.PrivateForkWorkflows{RunWorkflowsFromForkPullRequests: ptr(true), SendWriteTokensToWorkflows: ptr(true)}}}
	p := Build("o", "r", d, s, f, false)
	_, failed := p.Execute(context.Background())
	if len(failed) != 0 {
		t.Fatal(failed)
	}
	if len(p.Changes) != 2 {
		t.Fatalf("changes=%v", p.Changes)
	}
	body := f.bodies[0].(map[string]any)
	if body["run_workflows_from_fork_pull_requests"] != true || body["send_write_tokens_to_workflows"] != false {
		t.Fatalf("body=%v", body)
	}
}
func TestEnvironmentPartialUpdatePreservesProtection(t *testing.T) {
	reviewers := []config.EnvironmentReviewer{{Type: "Team", ID: 42}}
	vars := map[string]string{"REGION": "new"}
	d := &config.Config{Environments: &map[string]config.Environment{"production": {Variables: &vars}}}
	s := &github.State{Additional: config.Config{Environments: &map[string]config.Environment{"production": {WaitTimer: ptr(15), PreventSelfReview: ptr(true), Reviewers: &reviewers, DeploymentBranchPolicy: &config.DeploymentBranchPolicy{ProtectedBranches: true}, Variables: &map[string]string{"REGION": "old"}}}}}
	f := &recordingExec{}
	p := Build("o", "r", d, s, f, false)
	_, failed := p.Execute(context.Background())
	if len(failed) > 0 {
		t.Fatal(failed)
	}
	if f.environment.WaitTimer == nil || *f.environment.WaitTimer != 15 || f.environment.Reviewers == nil || len(*f.environment.Reviewers) != 1 || !f.environment.DeploymentBranchPolicy.ProtectedBranches {
		t.Fatalf("protection lost: %+v", f.environment)
	}
}
func TestCollectionReplacementStopsAfterFailedDelete(t *testing.T) {
	d := &config.Config{Autolinks: &map[string]config.Autolink{"ENG-": {URLTemplate: "https://new/<num>"}}}
	s := &github.State{AutolinkIDs: map[string]int64{"ENG-": 7}, Additional: config.Config{Autolinks: &map[string]config.Autolink{"ENG-": {URLTemplate: "https://old/<num>"}}}}
	f := &recordingExec{fakeExec: fakeExec{fail: map[string]bool{"DELETE /autolinks/7": true}}}
	p := Build("o", "r", d, s, f, false)
	if len(p.Changes) != 1 || p.Changes[0].Operation != Replace {
		t.Fatalf("changes=%+v", p.Changes)
	}
	_, failed := p.Execute(context.Background())
	if len(failed) != 1 || len(f.calls) != 1 {
		t.Fatalf("failed=%v calls=%v", failed, f.calls)
	}
}
func TestAdditionalIdempotenceAndEmptyCollections(t *testing.T) {
	d := &config.Config{DeployKeys: &map[string]config.DeployKey{"test": {Key: "ssh-ed25519 AAAA comment", ReadOnly: true}}, Actions: &config.ActionsSettings{Variables: &map[string]string{"region": "west"}}}
	s := &github.State{Actions: &config.ActionsSettings{Variables: &map[string]string{"REGION": "west"}}, Additional: config.Config{DeployKeys: &map[string]config.DeployKey{"test": {Key: "ssh-ed25519 AAAA", ReadOnly: true}}}}
	p := Build("o", "r", d, s, &fakeExec{}, false)
	if p.Drift {
		t.Fatalf("unexpected drift=%v", p.Changes)
	}
	d.Actions.Variables = &map[string]string{}
	p = Build("o", "r", d, s, &fakeExec{}, false)
	if len(p.Changes) != 1 || p.Changes[0].Operation != Remove {
		t.Fatalf("changes=%v", p.Changes)
	}
}
func TestPagesCreateBeforeConfigureAndDisable(t *testing.T) {
	d := &config.Config{Pages: &config.PagesSettings{Enabled: ptr(true), BuildType: ptr("workflow"), CNAME: ptr("")}}
	s := &github.State{Additional: config.Config{Pages: &config.PagesSettings{Enabled: ptr(false)}}}
	f := &recordingExec{}
	p := Build("o", "r", d, s, f, false)
	_, failed := p.Execute(context.Background())
	if len(failed) > 0 || len(f.calls) != 2 || f.calls[0] != "POST /pages" || f.calls[1] != "PUT /pages" {
		t.Fatalf("calls=%v failed=%v", f.calls, failed)
	}
	if f.bodies[1].(map[string]any)["cname"] != nil {
		t.Fatal("empty CNAME not cleared with null")
	}
	d.Pages = &config.PagesSettings{Enabled: ptr(false)}
	s.Additional.Pages.Enabled = ptr(true)
	p = Build("o", "r", d, s, f, false)
	if len(p.Changes) != 1 || p.Changes[0].Operation != Remove {
		t.Fatalf("changes=%v", p.Changes)
	}
}

func TestNamesDifferingOnlyInCaseNeverDeleteResources(t *testing.T) {
	d := &config.Config{Labels: &map[string]config.Label{"BUG": {Color: "abcdef", Description: "bug"}}, Environments: &map[string]config.Environment{"PRODUCTION": {WaitTimer: ptr(10)}}}
	s := &github.State{Additional: config.Config{Labels: &map[string]config.Label{"bug": {Color: "ABCDEF", Description: "bug"}}, Environments: &map[string]config.Environment{"production": {WaitTimer: ptr(10)}}}}
	p := Build("o", "r", d, s, &fakeExec{}, false)
	if p.Drift {
		t.Fatalf("case-only difference causes drift: %v", p.Changes)
	}
}
