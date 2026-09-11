package config

import (
	"os"
	"strings"
	"testing"
)

func TestAdditionalValidation(t *testing.T) {
	for _, input := range []string{
		"actions:\n  cache:\n    max_size_gb: 0\n",
		"actions:\n  variables:\n    GITHUB_TOKEN: forbidden\n",
		"actions:\n  variables:\n    a: one\n    A: two\n",
		"pages:\n  enabled: false\n  cname: example.com\n",
		"pages:\n  source:\n    branch: main\n    path: /invalid\n",
		"labels:\n  bug:\n    color: red\n",
		"environments:\n  prod:\n    deployment_branch_policy:\n      protected_branches: true\n      custom_branch_policies: true\n",
	} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Errorf("accepted invalid config:\n%s", input)
		}
	}
}
func TestProjectPreservesNestedScope(t *testing.T) {
	yes, no := true, false
	days := 7
	scope := &Config{Actions: &ActionsSettings{OIDC: &OIDCSettings{UseDefault: &yes}}, Pages: &PagesSettings{Enabled: &yes}}
	live := &Config{Actions: &ActionsSettings{OIDC: &OIDCSettings{UseDefault: &no}, Cache: &CacheSettings{MaxRetentionDays: &days}}, Pages: &PagesSettings{Enabled: &no}}
	got := Project(scope, live)
	if got.Actions.Cache != nil || *got.Actions.OIDC.UseDefault || *got.Pages.Enabled {
		t.Fatalf("project=%+v", got)
	}
}

func TestDocumentedDeploymentExampleParses(t *testing.T) {
	b, err := os.ReadFile("../../docs/EXAMPLES.md")
	if err != nil {
		t.Fatal(err)
	}
	section := strings.Split(string(b), "## Deployment and Build Configuration")
	if len(section) != 2 {
		t.Fatal("example missing")
	}
	fenced := strings.Split(section[1], "```yaml\n")
	if len(fenced) != 2 {
		t.Fatal("YAML example missing")
	}
	example := strings.SplitN(fenced[1], "```", 2)[0]
	if _, err := Parse([]byte(example)); err != nil {
		t.Fatalf("documentation example: %v", err)
	}
}
