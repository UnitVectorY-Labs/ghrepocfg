package app

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

func TestParseRepo(t *testing.T) {
	for input, want := range map[string]string{"owner/repo": "owner/repo", "https://github.com/owner/repo.git": "owner/repo", "git@github.com:owner/repo.git": "owner/repo", "ssh://git@github.com/owner/repo.git": "owner/repo"} {
		o, r, err := parseRepo(input)
		if err != nil || o+"/"+r != want {
			t.Errorf("parseRepo(%q)=%s/%s,%v", input, o, r, err)
		}
	}
	for _, input := range []string{"repo", "https://gitlab.com/o/r", "a/b/c"} {
		if _, _, err := parseRepo(input); err == nil {
			t.Errorf("parseRepo(%q) unexpectedly succeeded", input)
		}
	}
}

func TestRepoFromNonOriginGitRemote(t *testing.T) {
	dir := t.TempDir()
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "remote", "add", "upstream", "git@github.com:acme/widgets.git").Run(); err != nil {
		t.Fatal(err)
	}
	owner, repo, err := repoFromGit(dir)
	if err != nil || owner != "acme" || repo != "widgets" {
		t.Fatalf("repoFromGit = %s/%s, %v", owner, repo, err)
	}
	if err := exec.Command("git", "-C", dir, "remote", "add", "origin", "https://github.com/acme/other.git").Run(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repoFromGit(dir); err == nil {
		t.Fatal("ambiguous remotes should fail")
	}
}

func TestRunHelpAndUnknown(t *testing.T) {
	var out, errout bytes.Buffer
	if code := Run([]string{"help"}, strings.NewReader(""), &out, &errout, "test version"); code != 0 || !strings.Contains(out.String(), "ghrepocfg export") {
		t.Fatalf("help code=%d out=%q", code, out.String())
	}
	out.Reset()
	if code := Run([]string{"wat"}, strings.NewReader(""), &out, &errout, "test"); code != 1 {
		t.Fatalf("unknown code=%d", code)
	}
}

func TestConfirmDefaultsNo(t *testing.T) {
	for input, want := range map[string]bool{"\n": false, "n\n": false, "y\n": true, "YES\n": true} {
		if got := confirm(strings.NewReader(input), &bytes.Buffer{}, 2); got != want {
			t.Errorf("confirm(%q)=%v", input, got)
		}
	}
}

func TestDiffConfig(t *testing.T) {
	a := &config.Config{Repository: &config.RepositorySettings{HasWiki: boolp(false)}}
	b := &config.Config{Repository: &config.RepositorySettings{HasWiki: boolp(true), HasIssues: boolp(true)}}
	changes := diffConfig(a, b)
	if len(changes) != 2 {
		t.Fatalf("changes=%#v", changes)
	}
}

func TestNewExportDiffReportsLeafAdditions(t *testing.T) {
	after := &config.Config{Repository: &config.RepositorySettings{HasWiki: boolp(true), HasIssues: boolp(false)}}
	changes := diffConfig(nil, after)
	if len(changes) != 2 || changes[0].Path != "repository.has_issues" || changes[1].Path != "repository.has_wiki" {
		t.Fatalf("changes = %#v", changes)
	}
}
func boolp(v bool) *bool { return &v }
