package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func parseSemantic(s string) (*Config, error) { return Parse([]byte(s)) }

func semanticJSON(t *testing.T, c *Config) string {
	t.Helper()
	v, err := SemanticTree(c)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSemanticTreePreservesCollectionPresence(t *testing.T) {
	managed, err := parseSemantic("repository:\n  topics: []\n")
	if err != nil {
		t.Fatal(err)
	}
	unmanaged, err := parseSemantic("repository:\n  has_wiki: true\n")
	if err != nil {
		t.Fatal(err)
	}
	if semanticJSON(t, managed) == semanticJSON(t, unmanaged) {
		t.Fatal("empty managed collection must differ from omitted collection")
	}
}

func TestSemanticTreeDomainNormalization(t *testing.T) {
	a, err := parseSemantic("repository:\n  topics: [z, a]\nactions:\n  variables:\n    foo: bar\ncustom_properties:\n  tags: [z, a]\nlabels:\n  L:\n    color: AABBCC\n")
	if err != nil {
		t.Fatal(err)
	}
	b, err := parseSemantic("repository:\n  topics: [a, z]\nactions:\n  variables:\n    FOO: bar\ncustom_properties:\n  tags: [a, z]\nlabels:\n  l:\n    color: aabbcc\n")
	if err != nil {
		t.Fatal(err)
	}
	if semanticJSON(t, a) != semanticJSON(t, b) {
		t.Fatal("equivalent unordered and case-insensitive values differ")
	}

	oidc1, err := parseSemantic("actions:\n  oidc:\n    include_claim_keys: [sub, repository]\n")
	if err != nil {
		t.Fatal(err)
	}
	oidc2, err := parseSemantic("actions:\n  oidc:\n    include_claim_keys: [repository, sub]\n")
	if err != nil {
		t.Fatal(err)
	}
	if semanticJSON(t, oidc1) == semanticJSON(t, oidc2) {
		t.Fatal("OIDC claim order is meaningful")
	}
}

func TestSemanticTreeRulesetDefaultsAndNilSlices(t *testing.T) {
	a, err := parseSemantic("rulesets:\n  main:\n    enforcement: active\n    rules:\n      - type: update\n        parameters:\n          update_allows_fetch_and_merge: false\n")
	if err != nil {
		t.Fatal(err)
	}
	b, err := parseSemantic("rulesets:\n  main:\n    enforcement: active\n    rules: []\n")
	if err != nil {
		t.Fatal(err)
	}
	// The rules themselves differ, but the model must represent a nil rules
	// slice and an empty rules slice identically when the containing ruleset is
	// otherwise equal.
	aRules := (*a.Rulesets)["main"]
	bRules := (*b.Rulesets)["main"]
	aRules.Rules = nil
	bRules.Rules = []Rule{}
	aRules.BypassActors = nil
	bRules.BypassActors = []BypassActor{}
	if !reflect.DeepEqual(semanticJSON(t, &Config{Rulesets: &map[string]Ruleset{"main": aRules}}), semanticJSON(t, &Config{Rulesets: &map[string]Ruleset{"main": bRules}})) {
		t.Fatal("nil and empty non-pointer slices should compare equally")
	}
}
