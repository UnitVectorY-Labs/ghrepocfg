package exportconfig

import (
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

func p[T any](v T) *T { return &v }
func TestScopedFromStatePreservesScalarAndSectionScope(t *testing.T) {
	base := &config.Config{Repository: &config.RepositorySettings{HasWiki: p(false)}, Collaborators: &map[string]config.Access{}}
	state := &github.State{Repository: &config.RepositorySettings{HasWiki: p(true), HasIssues: p(true)}, Collaborators: map[string]github.Collaborator{"alice": {Permission: "push"}}, Teams: map[string]github.Team{"platform": {Permission: "admin"}}}
	got := ScopedFromState(base, state)
	if got.Repository.HasWiki == nil || !*got.Repository.HasWiki {
		t.Fatal("managed value not refreshed")
	}
	if got.Repository.HasIssues != nil {
		t.Fatal("scope expanded to omitted field")
	}
	if got.Teams != nil {
		t.Fatal("scope expanded to omitted section")
	}
	if (*got.Collaborators)["alice"].Permission != "push" {
		t.Fatal("managed collection not refreshed authoritatively")
	}
}

func TestFullExportRoundTripsRulesWithOmittedDefaultParameters(t *testing.T) {
	state := &github.State{Rulesets: map[string]github.Ruleset{
		"tag": {Value: config.Ruleset{
			Target: "tag", Enforcement: "active", Rules: []config.Rule{{Type: "update"}},
		}},
	}}
	b, err := config.Marshal(FromState(state))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.Parse(b); err != nil {
		t.Fatalf("full export did not produce a valid configuration: %v\n%s", err, b)
	}
	if !strings.Contains(string(b), "- type: update") {
		t.Fatalf("export omitted update rule: %s", b)
	}
}

func TestCustomPropertiesExportFullAndScoped(t *testing.T) {
	properties := map[string]config.CustomPropertyValue{
		"status":    {Value: "active"},
		"platforms": {Value: []string{"linux", "macos"}},
	}
	state := &github.State{CustomProperties: properties}
	full := FromState(state)
	if full.CustomProperties == nil || (*full.CustomProperties)["status"].Value != "active" {
		t.Fatalf("full custom properties = %#v", full.CustomProperties)
	}
	base := &config.Config{CustomProperties: &map[string]config.CustomPropertyValue{}}
	scoped := ScopedFromState(base, state)
	if scoped.CustomProperties == nil || len(*scoped.CustomProperties) != 2 {
		t.Fatalf("scoped custom properties = %#v", scoped.CustomProperties)
	}
	original := properties["platforms"]
	original.Value.([]string)[0] = "changed"
	if got := (*scoped.CustomProperties)["platforms"].Value.([]string)[0]; got != "linux" {
		t.Fatalf("export did not clone array, got %q", got)
	}
}

func TestAdditionalSectionsFullAndScoped(t *testing.T) {
	state := &github.State{
		Repository: &config.RepositorySettings{ImmutableReleases: p(true)},
		Security:   &config.SecuritySettings{PrivateVulnerabilityReporting: p(true)},
		Actions:    &config.ActionsSettings{SHAPinningRequired: p(true), Cache: &config.CacheSettings{MaxRetentionDays: p(7), MaxSizeGB: p(10)}},
		Additional: config.Config{
			Pages:        &config.PagesSettings{Enabled: p(false)},
			Environments: &map[string]config.Environment{},
			Labels:       &map[string]config.Label{"bug": {Color: "d73a4a", Description: "Bug"}},
			Autolinks:    &map[string]config.Autolink{},
			DeployKeys:   &map[string]config.DeployKey{},
		},
	}
	full := FromState(state)
	b, err := config.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.Parse(b); err != nil {
		t.Fatalf("invalid export: %v\n%s", err, b)
	}
	base := &config.Config{Actions: &config.ActionsSettings{Cache: &config.CacheSettings{MaxRetentionDays: p(3)}}, Labels: &map[string]config.Label{}}
	scoped := ScopedFromState(base, state)
	if scoped.Actions.Cache.MaxSizeGB != nil || scoped.Actions.SHAPinningRequired != nil || scoped.Pages != nil || scoped.Labels == nil || len(*scoped.Labels) != 1 {
		t.Fatalf("scope expanded: %+v", scoped)
	}
}

func TestScopedEnvironmentPreservesChildFields(t *testing.T) {
	base := &config.Config{Environments: &map[string]config.Environment{"prod": {Variables: &map[string]string{}}}}
	state := &github.State{Additional: config.Config{Environments: &map[string]config.Environment{"prod": {WaitTimer: p(10), Variables: &map[string]string{"REGION": "west"}}}}}
	got := ScopedFromState(base, state)
	if (*got.Environments)["prod"].WaitTimer != nil || len(*(*got.Environments)["prod"].Variables) != 1 {
		t.Fatalf("scope expanded: %+v", *got.Environments)
	}
}

func TestUnavailableExportPreservesExistingValues(t *testing.T) {
	base := &config.Config{CustomProperties: &map[string]config.CustomPropertyValue{"team": {Value: "platform"}}, Repository: &config.RepositorySettings{Description: p("old")}}
	state := &github.State{Repository: &config.RepositorySettings{Description: p("new")}}
	state.Omit([]string{"custom_properties"}, "denied")
	if full := FromState(state); full.CustomProperties != nil {
		t.Fatal("unknown collection exported as empty")
	}
	scoped := ScopedFromState(base, state)
	if scoped.CustomProperties == nil || (*scoped.CustomProperties)["team"].Value != "platform" || *scoped.Repository.Description != "new" {
		t.Fatalf("scoped=%+v", scoped)
	}
}

func TestUnavailableEnvironmentVariablePreservesLiteralName(t *testing.T) {
	base := &config.Config{Environments: &map[string]config.Environment{"PROD.US": {WaitTimer: p(1), Variables: &map[string]string{"A": "keep"}}}}
	state := &github.State{Additional: config.Config{Environments: &map[string]config.Environment{"prod.us": {WaitTimer: p(2)}}}}
	state.Omit([]string{"environments", "prod.us", "variables"}, "denied")
	got := ScopedFromState(base, state)
	env := (*got.Environments)["prod.us"]
	if env.Variables == nil || (*env.Variables)["A"] != "keep" || *env.WaitTimer != 2 {
		t.Fatalf("environment=%+v", env)
	}
}

func TestAvailabilityCopyPreservesIntegerIDs(t *testing.T) {
	id := int64(9007199254740993)
	state := &github.State{Rulesets: map[string]github.Ruleset{"main": {Value: config.Ruleset{Enforcement: "active", BypassActors: []config.BypassActor{{ActorID: &id, ActorType: "Team"}}}}}}
	got := FromState(state)
	if actual := *(*got.Rulesets)["main"].BypassActors[0].ActorID; actual != id {
		t.Fatalf("actor ID changed: got %d want %d", actual, id)
	}
}
