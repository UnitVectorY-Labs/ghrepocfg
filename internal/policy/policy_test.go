package policy

import (
	"bytes"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

func layer(name, yaml string) Layer { return Layer{Name: name, Config: []byte(yaml)} }
func mustConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	c, err := config.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return c
}

func TestResolveMergesMapsScalarsAndReplacesArrays(t *testing.T) {
	got, err := Resolve([]Layer{
		layer("base", "repository:\n  has_wiki: false\n  description: base\n  topics: [one, two]\nactions:\n  enabled: true\n"),
		layer("team", "repository:\n  description: team\n  topics: [team]\nactions:\n  default_workflow_permissions: read\n"),
		layer("repo", "repository:\n  has_wiki: true\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	c := mustConfig(t, string(got))
	if c.Repository == nil || c.Repository.HasWiki == nil || !*c.Repository.HasWiki || *c.Repository.Description != "team" || len(*c.Repository.Topics) != 1 || (*c.Repository.Topics)[0] != "team" {
		t.Fatalf("unexpected resolved config: %#v", c.Repository)
	}
	if c.Actions == nil || c.Actions.Enabled == nil || !*c.Actions.Enabled || *c.Actions.DefaultWorkflowPermissions != "read" {
		t.Fatalf("nested inheritance lost: %#v", c.Actions)
	}
}

func TestResolveManyLayersAndExplicitEmptyPresence(t *testing.T) {
	layers := []Layer{layer("0", "repository:\n  topics: [zero]\n")}
	for i := 1; i < 12; i++ {
		layers = append(layers, layer(string(rune('a'+i)), "repository:\n  description: layer\n"))
	}
	layers = append(layers, layer("empty", "repository:\n  topics: []\n"))
	got, err := Resolve(layers)
	if err != nil {
		t.Fatal(err)
	}
	c := mustConfig(t, string(got))
	if c.Repository.Topics == nil || len(*c.Repository.Topics) != 0 {
		t.Fatalf("explicit empty array not preserved: %#v", c.Repository.Topics)
	}
	if c.Repository.Description == nil {
		t.Fatal("inherited fields disappeared")
	}
}

func TestResolveNullInheritsExceptCustomPropertyNull(t *testing.T) {
	got, err := Resolve([]Layer{layer("base", "repository:\n  description: inherited\ncustom_properties:\n  flag: value\n"), layer("lower", "repository:\n  description: null\ncustom_properties:\n  flag: null\n")})
	if err != nil {
		t.Fatal(err)
	}
	c := mustConfig(t, string(got))
	if *c.Repository.Description != "inherited" {
		t.Fatalf("null unexpectedly replaced scalar: %q", *c.Repository.Description)
	}
	if c.CustomProperties == nil || (*c.CustomProperties)["flag"].Value != nil {
		t.Fatalf("custom property null not retained: %#v", c.CustomProperties)
	}
}

func TestResolveExactLockProtectsParentReplacement(t *testing.T) {
	constraints := []byte("locks:\n  - path: /actions/default_workflow_permissions\n    mode: exact\n")
	layers := []Layer{layer("base", "actions:\n  default_workflow_permissions: read\n  enabled: true\n")}
	layers[0].Constraints = constraints
	layers = append(layers, layer("lower", "actions:\n  default_workflow_permissions: write\n  enabled: false\n"))
	if _, err := Resolve(layers); err == nil || !strings.Contains(err.Error(), "default_workflow_permissions") || !strings.Contains(err.Error(), "base") {
		t.Fatalf("expected useful lock violation, got %v", err)
	}
}

func TestResolveContainsLocksScalarAndStructuredArrays(t *testing.T) {
	base := layer("base", "repository:\n  topics: [security-reviewed]\nactions:\n  selected_actions:\n    patterns_allowed:\n      - old/action@*\n")
	base.Constraints = []byte("locks:\n  - path: /repository/topics\n    mode: contains\n  - path: /actions/selected_actions/patterns_allowed\n    mode: contains\n")
	good := layer("lower", "repository:\n  topics: [security-reviewed, payments]\nactions:\n  selected_actions:\n    patterns_allowed:\n      - old/action@*\n      - new/action@*\n")
	if _, err := Resolve([]Layer{base, good}); err != nil {
		t.Fatalf("allowed additions rejected: %v", err)
	}
	bad := layer("bad", "repository:\n  topics: [payments]\nactions:\n  selected_actions:\n    patterns_allowed: [new/action@*]\n")
	if _, err := Resolve([]Layer{base, bad}); err == nil || !strings.Contains(err.Error(), "repository/topics") {
		t.Fatalf("missing required element accepted: %v", err)
	}
}

func TestConstraintValidation(t *testing.T) {
	for _, data := range []string{"locks:\n  - path: actions/x\n    mode: exact\n", "locks:\n  - path: /repository/topics~2x\n    mode: exact\n"} {
		if _, err := ParseConstraints([]byte(data)); err == nil {
			t.Errorf("invalid constraints accepted: %q", data)
		}
	}
	wrongType := layer("base", "repository:\n  has_wiki: true\n")
	wrongType.Constraints = []byte("locks:\n  - path: /repository/has_wiki\n    mode: contains\n")
	if _, err := Resolve([]Layer{wrongType}); err == nil {
		t.Error("contains lock on scalar accepted")
	}
	if _, err := Resolve([]Layer{layer("base", "repository:\n  has_wiki: true\n")}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveDeterministic(t *testing.T) {
	layers := []Layer{layer("a", "repository:\n  has_wiki: true\n  topics: [z, a]\n"), layer("b", "repository:\n  description: x\n")}
	a, err := Resolve(layers)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve(layers)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("resolve output changed between runs:\n%s\n%s", a, b)
	}
}

func TestCompareSemanticAndPresence(t *testing.T) {
	a := mustConfig(t, "repository:\n  topics: [one, two]\n")
	b := mustConfig(t, "# comment\nrepository:\n  topics: [two, one]\n")
	d, err := Compare(a, b)
	if err != nil || d.Changed {
		t.Fatalf("unordered semantic equality failed: %#v %v", d, err)
	}
	c := mustConfig(t, "repository:\n  topics: []\n")
	d, err = Compare(a, c)
	if err != nil || !d.Changed || len(d.Changes) != 1 || d.Changes[0].Path != "/repository/topics" {
		t.Fatalf("presence/array diff failed: %#v %v", d, err)
	}
}

func TestCompareOrderedArrayChanges(t *testing.T) {
	a := mustConfig(t, "actions:\n  oidc:\n    include_claim_keys: [repository, workflow]\n")
	b := mustConfig(t, "actions:\n  oidc:\n    include_claim_keys: [workflow, repository]\n")
	d, err := Compare(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Changed {
		t.Fatal("ordered array reorder should change")
	}
}

func TestResolveExactArrayAndAccumulatedConstraints(t *testing.T) {
	base := layer("base", "repository:\n  topics: [security-reviewed]\n")
	base.Constraints = []byte("locks:\n  - path: /repository/topics\n    mode: exact\n")
	middle := layer("middle", "repository:\n  description: middle\n")
	middle.Constraints = []byte("locks:\n  - path: /repository/description\n    mode: exact\n")
	good := layer("good", "repository:\n  description: middle\n")
	if _, err := Resolve([]Layer{base, middle, good}); err != nil {
		t.Fatal(err)
	}
	bad := layer("bad", "repository:\n  topics: [changed]\n")
	if _, err := Resolve([]Layer{base, middle, bad}); err == nil || !strings.Contains(err.Error(), "base") {
		t.Fatalf("exact array replacement accepted: %v", err)
	}
	weak := layer("weakener", "repository:\n  topics: [security-reviewed]\n")
	weak.Constraints = []byte("locks: []\n")
	if _, err := Resolve([]Layer{base, weak, bad}); err == nil {
		t.Fatal("lower empty constraints weakened inherited lock")
	}
}

func TestResolveContainsLockRequiresStructuredArrayElement(t *testing.T) {
	base := layer("base", "rulesets:\n  policy:\n    enforcement: active\n    bypass_actors:\n      - actor_id: 42\n        actor_type: Team\n        bypass_mode: always\n")
	base.Constraints = []byte("locks:\n  - path: /rulesets/policy/bypass_actors\n    mode: contains\n")
	withExtra := layer("child", "rulesets:\n  policy:\n    enforcement: active\n    bypass_actors:\n      - actor_id: 42\n        actor_type: Team\n        bypass_mode: always\n      - actor_id: 7\n        actor_type: User\n        bypass_mode: pull_request\n")
	if _, err := Resolve([]Layer{base, withExtra}); err != nil {
		t.Fatalf("structured required element with addition rejected: %v", err)
	}
	changed := layer("changed", "rulesets:\n  policy:\n    enforcement: active\n    bypass_actors:\n      - actor_id: 42\n        actor_type: Team\n        bypass_mode: pull_request\n")
	if _, err := Resolve([]Layer{base, changed}); err == nil || !strings.Contains(err.Error(), "bypass_actors") {
		t.Fatalf("modified structured required element accepted: %v", err)
	}
}

func TestResolveExactObjectAndMapLocks(t *testing.T) {
	object := layer("object", "repository:\n  has_wiki: true\n  description: fixed\n")
	object.Constraints = []byte("locks:\n  - path: /repository\n    mode: exact\n")
	if _, err := Resolve([]Layer{object, layer("child", "repository:\n  has_wiki: false\n")}); err == nil || !strings.Contains(err.Error(), "/repository") {
		t.Fatalf("exact object replacement accepted: %v", err)
	}
	base := layer("map", "collaborators:\n  alice:\n    permission: push\n")
	base.Constraints = []byte("locks:\n  - path: /collaborators\n    mode: exact\n")
	if _, err := Resolve([]Layer{base, layer("child", "collaborators:\n  alice:\n    permission: push\n  bob:\n    permission: pull\n")}); err == nil || !strings.Contains(err.Error(), "/collaborators") {
		t.Fatalf("exact map extension accepted: %v", err)
	}
}
