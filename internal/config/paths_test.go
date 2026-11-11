package config

import (
	"encoding/json/v2"
	"strings"
	"testing"
)

func TestWithUnavailablePreservesEmptyManagedValuesAndLargeIDs(t *testing.T) {
	const id int64 = 9007199254740993
	empty := map[string]Label{}
	reviewers := []EnvironmentReviewer{{Type: "User", ID: id}}
	environments := map[string]Environment{"prod": {Reviewers: &reviewers}}
	c := &Config{Labels: &empty, Environments: &environments}
	got := WithUnavailable(c, nil, nil)
	if got.Labels == nil || len(*got.Labels) != 0 || got.Environments == nil || (*(*got.Environments)["prod"].Reviewers)[0].ID != id {
		t.Fatalf("round trip lost managed values: %#v", got)
	}
	b, err := json.Marshal(got)
	if err != nil || !strings.Contains(string(b), `"labels":{}`) {
		t.Fatalf("JSON = %s, error = %v", b, err)
	}
}
