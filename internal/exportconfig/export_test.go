package exportconfig

import (
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
	"testing"
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
