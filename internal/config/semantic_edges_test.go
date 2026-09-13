package config

import "testing"

func TestSemanticFieldMatrix(t *testing.T) {
	tests := []struct {
		name, a, b string
		equal      bool
	}{
		{"ruleset defaults", "rulesets:\n  main:\n    enforcement: active\n    rules: [{type: update}]\n", "rulesets:\n  main:\n    target: branch\n    enforcement: active\n    bypass_actors: []\n    rules: [{type: update, parameters: {update_allows_fetch_and_merge: false}}]\n", true},
		{"rules order", "rulesets:\n  main:\n    enforcement: active\n    rules: [{type: deletion}, {type: creation}]\n", "rulesets:\n  main:\n    enforcement: active\n    rules: [{type: creation}, {type: deletion}]\n", true},
		{"nested rule selections", "rulesets:\n  main:\n    enforcement: active\n    rules: [{type: required_status_checks, parameters: {required_status_checks: [{context: z}, {context: a}]}}]\n", "rulesets:\n  main:\n    enforcement: active\n    rules: [{type: required_status_checks, parameters: {required_status_checks: [{context: a}, {context: z}]}}]\n", true},
		{"map punctuation", "environments:\n  'x.y/~':\n    variables: {foo: a}\n    reviewers: [{type: User, id: 2}, {type: Team, id: 1}]\n", "environments:\n  'X.Y/~':\n    variables: {FOO: a}\n    reviewers: [{type: Team, id: 1}, {type: User, id: 2}]\n", true},
		{"key comments", "deploy_keys:\n  build: {key: 'ssh-ed25519 AAAA first', read_only: true}\n", "deploy_keys:\n  build: {key: 'ssh-ed25519 AAAA second', read_only: true}\n", true},
		{"empty authoritative map", "repository: {}\n", "repository: {}\nlabels: {}\n", false},
		{"null custom property presence", "custom_properties: {}\n", "custom_properties: {flag: null}\n", false},
		{"empty strings", "repository: {}\n", "repository: {description: ''}\n", false},
		{"false", "repository: {}\n", "repository: {has_wiki: false}\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := Parse([]byte(tt.a))
			if err != nil {
				t.Fatal(err)
			}
			b, err := Parse([]byte(tt.b))
			if err != nil {
				t.Fatal(err)
			}
			if got := semanticJSON(t, a) == semanticJSON(t, b); got != tt.equal {
				t.Fatalf("equality=%t want %t\na=%s\nb=%s", got, tt.equal, semanticJSON(t, a), semanticJSON(t, b))
			}
		})
	}
}
