package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveCLIIsOfflineAndSupportsOrderedLayers(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", filepath.Join(t.TempDir(), "empty"))
	dir := t.TempDir()
	base := writeFile(t, dir, "base.yaml", "repository:\n  has_wiki: false\n")
	child := writeFile(t, dir, "child.yaml", "repository:\n  has_wiki: true\n")
	var out, errout bytes.Buffer
	if code := Run([]string{"resolve", "--layer", base, "--layer", child}, strings.NewReader(""), &out, &errout, "test"); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errout.String())
	}
	if !strings.Contains(out.String(), "has_wiki: true") {
		t.Fatalf("resolved output=%q", out.String())
	}
	if strings.Contains(errout.String(), "token") || strings.Contains(errout.String(), "GitHub") || strings.Contains(errout.String(), "auth") {
		t.Fatalf("offline command attempted auth: %q", errout.String())
	}
}

func TestResolveCLIConstraintAssociationAndArgumentErrors(t *testing.T) {
	dir := t.TempDir()
	base := writeFile(t, dir, "base.yaml", "repository:\n  has_wiki: true\n")
	constraints := writeFile(t, dir, "policy.yaml", "locks:\n  - path: /repository/has_wiki\n    mode: exact\n")
	child := writeFile(t, dir, "child.yaml", "repository:\n  has_wiki: true\n")
	var out, errout bytes.Buffer
	if code := Run([]string{"resolve", "--constraints", constraints, "--layer", base}, nil, &out, &errout, "test"); code != exitError || out.Len() != 0 {
		t.Fatalf("constraints before layer accepted: code=%d out=%q err=%q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"resolve", "--layer", base, "--constraints", constraints, "--layer", child}, nil, &out, &errout, "test"); code != exitOK || !strings.Contains(out.String(), "has_wiki: true") {
		t.Fatalf("associated constraints failed: code=%d out=%q err=%q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"resolve", "--help"}, nil, &out, &errout, "test"); code != exitOK || !strings.Contains(out.String()+errout.String(), "--constraints") {
		t.Fatalf("resolve help missing constraints: code=%d out=%q err=%q", code, out.String(), errout.String())
	}
}

func TestResolveCLIOutputUnchangedWhenValidationFails(t *testing.T) {
	dir := t.TempDir()
	good := writeFile(t, dir, "good.yaml", "repository:\n  has_wiki: true\n")
	bad := writeFile(t, dir, "bad.yaml", "repository:\n  unknown_key: true\n")
	dest := writeFile(t, dir, "effective.yaml", "sentinel\n")
	var out, errout bytes.Buffer
	if code := Run([]string{"resolve", "--layer", good, "--layer", bad, "--output", dest}, nil, &out, &errout, "test"); code != exitError {
		t.Fatalf("code=%d stderr=%q", code, errout.String())
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "sentinel\n" {
		t.Fatalf("failed resolve modified output: %q", b)
	}
}

func TestDiffCLIExitCodesAndJSON(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", filepath.Join(t.TempDir(), "empty"))
	dir := t.TempDir()
	a := writeFile(t, dir, "a.yaml", "repository:\n  has_wiki: false\n")
	b := writeFile(t, dir, "b.yaml", "# formatting\nrepository:\n  has_wiki: false\n")
	c := writeFile(t, dir, "c.yaml", "repository:\n  has_wiki: true\n")
	var out, errout bytes.Buffer
	if code := Run([]string{"diff", a, b}, nil, &out, &errout, "test"); code != exitOK || !strings.Contains(out.String(), "No effective") {
		t.Fatalf("equal diff code=%d out=%q err=%q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"diff", a, c}, nil, &out, &errout, "test"); code != exitDrift || !strings.Contains(out.String(), "/repository/has_wiki") {
		t.Fatalf("changed diff code=%d out=%q", code, out.String())
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"diff", "missing", c}, nil, &out, &errout, "test"); code != exitError || out.Len() != 0 {
		t.Fatalf("invalid diff code=%d out=%q", code, out.String())
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"diff", "--json", a, c}, nil, &out, &errout, "test"); code != exitDrift {
		t.Fatalf("json changed code=%d err=%q", code, errout.String())
	}
	var result struct {
		Changed bool `json:"changed"`
		Changes []struct {
			Path, Operation string
			Before, After   json.RawMessage
		} `json:"changes"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON %q: %v", out.String(), err)
	}
	if !result.Changed || len(result.Changes) != 1 || result.Changes[0].Path != "/repository/has_wiki" || result.Changes[0].Operation != "modify" {
		t.Fatalf("unexpected JSON %#v", result)
	}
}

func TestDiffCLIReportsManagementBoundaryAndArrayChanges(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.yaml", "repository:\n  has_wiki: false\n")
	b := writeFile(t, dir, "b.yaml", "repository:\n  has_wiki: false\n  topics: []\n")
	var out, errout bytes.Buffer
	if code := Run([]string{"diff", a, b}, nil, &out, &errout, "test"); code != exitDrift || !strings.Contains(out.String(), "/repository/topics") {
		t.Fatalf("managed empty collection not reported: code=%d out=%q err=%q", code, out.String(), errout.String())
	}
}

func TestDiffCLIRejectsInvalidConfiguration(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.yaml", "repository:\n  has_wiki: true\n")
	b := writeFile(t, dir, "bad.yaml", "repository:\n  typo: true\n")
	var out, errout bytes.Buffer
	if code := Run([]string{"diff", a, b}, nil, &out, &errout, "test"); code != exitError {
		t.Fatalf("code=%d stderr=%q", code, errout.String())
	}
}
