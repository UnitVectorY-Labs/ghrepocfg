package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string, headers http.Header) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: headers}
}
func testClient(fn roundTripFunc) *Client {
	return &Client{http: &http.Client{Transport: fn}, token: "secret", base: "https://api.github.test"}
}

func TestRequestHeadersAndError(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Errorf("version = %q", r.Header.Get("X-GitHub-Api-Version"))
		}
		return response(http.StatusForbidden, `{"message":"policy blocked it"}`, make(http.Header)), nil
	})
	_, err := c.request(context.Background(), http.MethodGet, "/test", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "policy blocked it") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPagedFollowsLink(t *testing.T) {
	calls := 0
	c := testClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			h := make(http.Header)
			h.Set("Link", `<https://api.github.test/items?page=2>; rel="next"`)
			return response(http.StatusOK, `[{"id":1}]`, h), nil
		}
		return response(http.StatusOK, `[{"id":2}]`, make(http.Header)), nil
	})
	var got []struct {
		ID int `json:"id"`
	}
	if err := c.paged(context.Background(), "/items", &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("paged result = %#v", got)
	}
}

func TestMutationRequestShapes(t *testing.T) {
	type seen struct {
		Method, Path string
		Body         map[string]any
	}
	requests := make(chan seen, 3)
	c := testClient(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		requests <- seen{r.Method, r.URL.EscapedPath(), body}
		return response(http.StatusNoContent, "", make(http.Header)), nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.SetCollaborator(ctx, "acme", "repo", "Octo Cat", "push"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetTeam(ctx, "acme", "repo", "platform", "maintain"); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveRuleset(ctx, "acme", "repo", 42); err != nil {
		t.Fatal(err)
	}
	one, two, three := <-requests, <-requests, <-requests
	if one.Method != "PUT" || one.Path != "/repos/acme/repo/collaborators/Octo%20Cat" || one.Body["permission"] != "push" {
		t.Fatalf("collaborator request = %#v", one)
	}
	if two.Path != "/orgs/acme/teams/platform/repos/acme/repo" || two.Body["permission"] != "maintain" {
		t.Fatalf("team request = %#v", two)
	}
	if three.Method != "DELETE" || three.Path != "/repos/acme/repo/rulesets/42" {
		t.Fatalf("ruleset request = %#v", three)
	}
}

func TestRulesetBodyUsesEmptyArraysNotNull(t *testing.T) {
	body := rulesetBody("main", config.Ruleset{Enforcement: "disabled"})
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, `"bypass_actors":[]`) || !strings.Contains(text, `"rules":[]`) || strings.Contains(text, `"conditions"`) {
		t.Fatalf("ruleset body = %s", text)
	}
}

func TestReplaceTopicsSendsEmptyArray(t *testing.T) {
	var raw string
	c := testClient(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
		return response(http.StatusOK, `{"names":[]}`, make(http.Header)), nil
	})
	if err := c.ReplaceTopics(context.Background(), "o", "r", []string{}); err != nil {
		t.Fatal(err)
	}
	if raw != `{"names":[]}` {
		t.Fatalf("body = %s", raw)
	}
}

func TestPermissionNormalization(t *testing.T) {
	for in, want := range map[string]string{"read": "pull", "write": "push", "maintain": "maintain", "security-manager": "custom:security-manager"} {
		if got := permission(in, nil); got != want {
			t.Errorf("permission(%q)=%q want %q", in, got, want)
		}
	}
}

func TestReadRequiresAdminForAuthoritativeRulesets(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		return response(http.StatusOK, `{"permissions":{"admin":false}}`, make(http.Header)), nil
	})
	_, err := c.Read(context.Background(), "acme", "repo", ReadScope{Rulesets: true})
	if err == nil || !strings.Contains(err.Error(), "repository admin access is required") {
		t.Fatalf("Read() error = %v", err)
	}
}
