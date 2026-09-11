package github

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

func TestSecurityUpdatesDecodeDisabledBody(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "automated-security-fixes") {
			return response(200, `{"enabled":false,"paused":true}`, nil), nil
		}
		return response(204, "", nil), nil
	})
	got, err := c.readSecurity(context.Background(), "o", "r", nil)
	if err != nil || got.AutomatedSecurityFixes == nil || *got.AutomatedSecurityFixes {
		t.Fatalf("state=%+v err=%v", got, err)
	}
}
func TestCollaboratorCustomRoleAndTeamProvenance(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(r.URL.Path, "collaborators"):
			return response(200, `[{"login":"alice","role_name":"release-manager","permissions":{"push":true}}]`, nil), nil
		case strings.HasSuffix(r.URL.Path, "teams"):
			return response(200, `[{"slug":"direct","permission":"push","type":"organization","access_source":"direct"},{"slug":"inherited","permission":"admin","access_source":"organization"},{"slug":"enterprise","permission":"push","type":"enterprise"}]`, nil), nil
		}
		return response(200, `[]`, nil), nil
	})
	collabs, err := c.readCollaborators(context.Background(), "o", "r")
	if err != nil || collabs["alice"].Permission != "custom:release-manager" {
		t.Fatalf("%v %v", collabs, err)
	}
	teams, err := c.readTeams(context.Background(), "o", "r")
	if err != nil || len(teams) != 1 || teams["direct"].Permission != "push" {
		t.Fatalf("%v %v", teams, err)
	}
}
func TestAdditionalReadScopeAndUnavailable(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/repos/o/r":
			return response(200, `{"permissions":{"admin":true}}`, nil), nil
		case "/repos/o/r/pages":
			return response(404, `{"message":"Not Found"}`, nil), nil
		case "/repos/o/r/branches", "/repos/o/r/tags/protection":
			return response(200, `[]`, nil), nil
		}
		t.Fatalf("unexpected request %s", r.URL)
		return nil, nil
	})
	enabled := true
	s, err := c.Read(context.Background(), "o", "r", ReadScope{Desired: &config.Config{Pages: &config.PagesSettings{Enabled: &enabled}}})
	if err != nil || s.Additional.Pages == nil || *s.Additional.Pages.Enabled {
		t.Fatalf("%+v %v", s, err)
	}
	// Explicit configuration never swallows unavailable endpoints.
	c = testClient(func(r *http.Request) (*http.Response, error) {
		return response(403, `{"message":"unavailable"}`, nil), nil
	})
	err = c.readAdditional(context.Background(), "o", "r", ReadScope{Desired: &config.Config{Labels: &map[string]config.Label{}}}, &State{})
	if err == nil {
		t.Fatal("explicit read silently omitted inaccessible labels")
	}
	// Full discovery omits unavailable groups without inventing empty collections.
	s = &State{Repository: &config.RepositorySettings{}, Security: &config.SecuritySettings{}, Actions: &config.ActionsSettings{}}
	err = c.readAdditional(context.Background(), "o", "r", ReadScope{Full: true}, s)
	if err != nil || s.Additional.Labels != nil || len(s.Warnings) == 0 {
		t.Fatalf("state=%+v err=%v", s, err)
	}
}
func TestWrappedVariablesPagination(t *testing.T) {
	calls := 0
	c := testClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(200, `{"variables":[{"name":"ONE","value":"1"}]}`, http.Header{"Link": []string{`<https://api.github.test/variables?page=2>; rel="next"`}}), nil
		}
		return response(200, `{"variables":[{"name":"TWO","value":"2"}]}`, nil), nil
	})
	vars, err := c.readVariables(context.Background(), "/variables")
	if err != nil || len(*vars) != 2 || (*vars)["TWO"] != "2" {
		t.Fatalf("%v %v", vars, err)
	}
}
func TestEnvironmentPoliciesAndVariables(t *testing.T) {
	calls := []string{}
	c := testClient(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Method == "GET" {
			if strings.HasSuffix(r.URL.Path, "deployment-branch-policies") {
				return response(200, `{"branch_policies":[{"id":1,"name":"old","type":"branch"},{"id":2,"name":"v*","type":"tag"}]}`, nil), nil
			}
			return response(200, `{"variables":[{"name":"OLD","value":"old"}]}`, nil), nil
		}
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		if r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "prod") {
			if body["deployment_branch_policy"] == nil {
				t.Fatal("missing policy")
			}
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "deployment-branch-policies") {
			if body["name"] != "main" || body["type"] != "branch" {
				t.Fatalf("body=%v", body)
			}
		}
		return response(204, "", nil), nil
	})
	branches := []string{"main"}
	vars := map[string]string{"NEW": "new"}
	err := c.SetEnvironment(context.Background(), "o", "r", "prod", config.Environment{DeploymentBranchPolicy: &config.DeploymentBranchPolicy{CustomBranchPolicies: true}, DeploymentBranchPatterns: &branches, Variables: &vars})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(calls, "\n")
	if strings.Contains(joined, "DELETE /repos/o/r/environments/prod/deployment-branch-policies/2") || !strings.Contains(joined, "DELETE /repos/o/r/environments/prod/deployment-branch-policies/1") || !strings.Contains(joined, "DELETE /repos/o/r/environments/prod/variables/OLD") {
		t.Fatalf("calls=%s", joined)
	}
}

func TestCacheWriteVerifiesEffectiveValue(t *testing.T) {
	for _, effective := range []string{`{"max_cache_retention_days":7}`, `{"max_cache_retention_days":3}`} {
		calls := 0
		c := testClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method == "PUT" {
				return response(204, "", nil), nil
			}
			return response(200, effective, nil), nil
		})
		err := c.Mutate(context.Background(), "o", "r", "PUT", "/actions/cache/retention-limit", map[string]any{"max_cache_retention_days": 3})
		if calls != 2 || (strings.Contains(effective, ":7") && err == nil) || (strings.Contains(effective, ":3") && err != nil) {
			t.Fatalf("effective=%s calls=%d err=%v", effective, calls, err)
		}
	}
}

func TestDisabledCodeScanningExportsOnlyState(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/repos/o/r/code-scanning/default-setup" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		return response(200, `{"state":"not-configured","languages":[],"query_suite":"default","runner_type":"standard","schedule":"weekly"}`, nil), nil
	})
	s := &State{Security: &config.SecuritySettings{}}
	err := c.readAdditional(context.Background(), "o", "r", ReadScope{Desired: &config.Config{Security: &config.SecuritySettings{CodeScanningDefaultSetup: &config.CodeScanningSetup{}}}}, s)
	if err != nil {
		t.Fatal(err)
	}
	setup := s.Security.CodeScanningDefaultSetup
	if setup == nil || *setup.State != "not-configured" || setup.Languages != nil || setup.RunnerType != nil {
		t.Fatalf("setup=%+v", setup)
	}
	if err := (&config.Config{Security: s.Security}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestEnvironmentPatternsRequireEffectiveCustomPolicy(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) { t.Fatal("must fail before writing"); return nil, nil })
	patterns := []string{"main"}
	err := c.SetEnvironment(context.Background(), "o", "r", "prod", config.Environment{DeploymentBranchPolicy: &config.DeploymentBranchPolicy{}, DeploymentBranchPatterns: &patterns})
	if err == nil {
		t.Fatal("patterns silently ignored")
	}
}
