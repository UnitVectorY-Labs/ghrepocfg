package policy

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveCanonicalEquivalentInputs(t *testing.T) {
	a, err := Resolve([]Layer{layer("a", "repository: {topics: [z, a]}\nlabels: {BUG: {color: AABBCC}}\n")})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve([]Layer{layer("b", "labels: {bug: {color: aabbcc}}\nrepository: {topics: [a, z]}\n")})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("canonical output differs:\n%s\n%s", a, b)
	}
}
func TestResolveIdentityAndEscapedLock(t *testing.T) {
	base := layer("base", "environments:\n  'Prod.X/~':\n    variables: {Foo: inherited}\n")
	base.Constraints = []byte("locks:\n - path: /environments/Prod.X~1~0/variables/Foo\n   mode: exact\n")
	lower := layer("lower", "environments:\n  'prod.x/~':\n    variables: {FOO: changed}\n")
	if _, err := Resolve([]Layer{base, lower}); err == nil || !strings.Contains(err.Error(), "constraint violation") {
		t.Fatalf("escaped case-insensitive lock: %v", err)
	}
	base.Constraints = nil
	out, err := Resolve([]Layer{base, lower})
	if err != nil {
		t.Fatal(err)
	}
	c := mustConfig(t, string(out))
	if len(*c.Environments) != 1 || (*(*c.Environments)["prod.x/~"].Variables)["FOO"] != "changed" {
		t.Fatalf("identity override failed: %s", out)
	}
}
func TestNullNonPointerRetainsLiteralSemantics(t *testing.T) {
	out, err := Resolve([]Layer{layer("base", "labels: {bug: {color: aabbcc, description: inherited}}\n"), layer("lower", "labels: {bug: {color: aabbcc, description: null}}\n")})
	if err != nil {
		t.Fatal(err)
	}
	c := mustConfig(t, string(out))
	if (*c.Labels)["bug"].Description != "" {
		t.Fatalf("null description failed to override: %s", out)
	}
}
func TestExplicitDefaultDescendantLock(t *testing.T) {
	base := layer("base", "rulesets:\n  main:\n    enforcement: active\n    rules: [{type: update, parameters: {update_allows_fetch_and_merge: false}}]\n")
	base.Constraints = []byte("locks:\n - path: /rulesets/main/rules/0/parameters/update_allows_fetch_and_merge\n   mode: exact\n")
	if _, err := Resolve([]Layer{base}); err != nil {
		t.Fatal(err)
	}
	lower := layer("lower", "rulesets:\n  main:\n    enforcement: active\n    rules: [{type: update, parameters: {update_allows_fetch_and_merge: true}}]\n")
	if _, err := Resolve([]Layer{base, lower}); err == nil {
		t.Fatal("default descendant lock bypassed")
	}
}
func TestConstraintRequiredPathAndRoot(t *testing.T) {
	for _, s := range []string{"locks: [{mode: exact}]", "locks: [{path: null, mode: exact}]", "locks: [{path: /repository, mode: exact, extra: true}]", "locks: []\n---\nlocks: []"} {
		if _, err := ParseConstraints([]byte(s)); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	base := layer("base", "repository: {has_wiki: false}\n")
	base.Constraints = []byte("locks: [{path: '', mode: exact}]")
	if _, err := Resolve([]Layer{base, layer("lower", "actions: {enabled: true}\n")}); err == nil {
		t.Fatal("root lock bypassed")
	}
}

func TestResolveAuthoredYAMLSemantics(t *testing.T) {
	base := layer("base", "labels:\n  123: {color: aabbcc}\n  other: {color: aabbcc}\nrepository: {description: 0x10, homepage: 2020-01-01}\n")
	lower := layer("lower", "labels:\n  123: {color: ffffff}\n")
	out, err := Resolve([]Layer{base, lower})
	if err != nil {
		t.Fatal(err)
	}
	c := mustConfig(t, string(out))
	if len(*c.Labels) != 2 || (*c.Labels)["123"].Color != "ffffff" || *c.Repository.Description != "0x10" || *c.Repository.Homepage != "2020-01-01" {
		t.Fatalf("literal YAML meaning changed: %s", out)
	}
	again, err := Resolve([]Layer{layer("effective", string(out))})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, again) {
		t.Fatalf("resolve not idempotent:\n%s\n%s", out, again)
	}
}
