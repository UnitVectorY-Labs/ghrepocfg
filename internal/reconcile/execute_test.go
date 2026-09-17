package reconcile

import (
	"context"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

func TestDeniedWriteSkipsDependentsButContinuesIndependentChanges(t *testing.T) {
	for _, strict := range []bool{false, true} {
		calls := []string{}
		change := func(path string, err error) Change {
			return Change{Path: path, apply: func(context.Context) error { calls = append(calls, path); return err }}
		}
		plan := &Plan{Complete: true, Changes: []Change{change("actions.permissions", &github.APIError{Status: 403}), change("actions.selected_actions", nil), change("repository.description", nil)}}
		result := plan.ExecuteWithPolicy(context.Background(), strict)
		if len(result.Failed) != 0 {
			t.Fatal(result.Failed)
		}
		if strict {
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		} else if len(calls) != 2 || len(result.Applied) != 1 || len(result.Skipped) != 2 {
			t.Fatalf("calls=%v result=%+v", calls, result)
		}
	}
}
func TestUnknownCollectionNeverPlansWrites(t *testing.T) {
	desired := &config.Config{CustomProperties: &map[string]config.CustomPropertyValue{"x": {Value: "y"}}}
	state := &github.State{}
	state.Omit([]string{"custom_properties"}, "denied")
	plan := Build("o", "r", desired, state, &fakeExec{}, false)
	if plan.Complete || plan.Drift || len(plan.Changes) != 0 {
		t.Fatalf("plan=%+v", plan)
	}
}
func TestPartialWriteRemainsFailure(t *testing.T) {
	plan := &Plan{Changes: []Change{{Path: "autolinks.x", apply: func(context.Context) error { return github.AfterMutation(&github.APIError{Status: 403}) }}}}
	result := plan.ExecuteWithPolicy(context.Background(), false)
	if len(result.Failed) != 1 || len(result.Skipped) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestEnvironmentDependenciesUseExactNames(t *testing.T) {
	calls := []string{}
	operation := func(path string, err error) Change {
		return Change{Operation: Add, Path: path, apply: func(context.Context) error { calls = append(calls, path); return err }}
	}
	p := &Plan{dependencies: map[string]string{"environments.prod.variables.X": "environments.prod"}, Changes: []Change{
		operation("environments.prod", &github.APIError{Status: 403}),
		operation("environments.prod.variables.X", nil),
		operation("environments.prod.variables.other.variables.X", nil),
	}}
	r := p.ExecuteWithPolicy(context.Background(), false)
	if len(calls) != 2 || len(r.Skipped) != 2 || len(r.Applied) != 1 {
		t.Fatalf("calls=%v result=%+v", calls, r)
	}
}
