package github

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

func TestPermissionErrorsAreNotRateLimitsOrValidation(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     int
		headers    http.Header
		message    string
		read, skip bool
	}{
		{name: "denied", status: 403, skip: true},
		{name: "hidden read", status: 404, read: true, skip: true},
		{name: "missing write", status: 404},
		{name: "rate limit", status: 403, headers: http.Header{"X-Ratelimit-Remaining": []string{"0"}}},
		{name: "secondary limit", status: 403, message: "You have exceeded a secondary rate limit"},
		{name: "validation", status: 422}, {name: "conflict", status: 409}, {name: "unauthorized", status: 401}, {name: "server", status: 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := UnavailableReason(&APIError{Status: tt.status, Headers: tt.headers, Message: tt.message}, tt.read)
			if ok != tt.skip {
				t.Fatalf("skip=%v want %v", ok, tt.skip)
			}
		})
	}
	if _, ok := UnavailableReason(AfterMutation(&APIError{Status: 403}), false); ok {
		t.Fatal("partial mutation was skipped")
	}
}

func TestScopedVariablesDoNotRequireActionsPolicy(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/repos/o/r":
			return response(200, `{"permissions":{"admin":false}}`, nil), nil
		case "/repos/o/r/actions/variables":
			return response(200, `{"variables":[{"name":"A","value":"b"}]}`, nil), nil
		case "/repos/o/r/branches", "/repos/o/r/tags/protection":
			return response(200, `[]`, nil), nil
		default:
			t.Fatalf("unnecessary endpoint: %s", r.URL)
			return nil, nil
		}
	})
	desired := &config.Config{Actions: &config.ActionsSettings{Variables: &map[string]string{}}}
	state, err := c.Read(context.Background(), "o", "r", ReadScope{Desired: desired, Actions: true})
	if err != nil || len(state.Unavailable) != 0 || (*state.Actions.Variables)["A"] != "b" {
		t.Fatalf("state=%+v error=%v", state, err)
	}
}

func TestHiddenRulesetBypassActorsAreUnavailable(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/repos/o/r":
			return response(200, `{}`, nil), nil
		case "/repos/o/r/rulesets":
			return response(200, `[{"id":1,"name":"main"}]`, nil), nil
		case "/repos/o/r/rulesets/1":
			return response(200, `{"id":1,"name":"main","enforcement":"active","rules":[]}`, nil), nil
		default:
			return response(200, `[]`, nil), nil
		}
	})
	state, err := c.Read(context.Background(), "o", "r", ReadScope{Rulesets: true})
	if err != nil || len(state.Unavailable) != 1 || state.Rulesets != nil {
		t.Fatalf("state=%+v error=%v", state, err)
	}
}

func TestCollectionPaginationFailureDoesNotExposePartialValues(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Path == "/repos/o/r":
			return response(200, `{}`, nil), nil
		case r.URL.Path == "/repos/o/r/labels" && r.URL.Query().Get("page") == "2":
			return response(403, `{"message":"denied"}`, nil), nil
		case r.URL.Path == "/repos/o/r/labels":
			return response(200, `[{"name":"bug","color":"ffffff"}]`, http.Header{"Link": []string{`<https://api.github.test/repos/o/r/labels?page=2>; rel="next"`}}), nil
		default:
			return response(200, `[]`, nil), nil
		}
	})
	state, err := c.Read(context.Background(), "o", "r", ReadScope{Desired: &config.Config{Labels: &map[string]config.Label{}}})
	if err != nil || state.Additional.Labels != nil || len(state.Unavailable) != 1 {
		t.Fatalf("state=%+v error=%v", state, err)
	}
}

func TestMissingScalarDoesNotBecomeZero(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/repos/o/r" || strings.HasSuffix(r.URL.Path, "/immutable-releases") {
			return response(200, `{}`, nil), nil
		}
		return response(200, `[]`, nil), nil
	})
	yes := true
	description := "new"
	desired := &config.Config{Repository: &config.RepositorySettings{ImmutableReleases: &yes, Description: &description}}
	state, err := c.Read(context.Background(), "o", "r", ReadScope{Desired: desired, Repository: true})
	if err != nil || len(state.Unavailable) != 2 || state.Repository.ImmutableReleases != nil {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}
