package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func mockAPI(t *testing.T, fn func(*http.Request) (int, string)) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("GH_TOKEN", "test")
	old := http.DefaultTransport
	http.DefaultTransport = transportFunc(func(r *http.Request) (*http.Response, error) {
		code, body := fn(r)
		return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = old })
}
func TestExportPermissionFailurePreservesFileAndStrictDoesNotWrite(t *testing.T) {
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.URL.Path == "/repos/o/r" {
			return 200, `{"description":"new"}`
		}
		if strings.HasSuffix(r.URL.Path, "/properties/values") {
			return 403, `{"message":"denied"}`
		}
		return 200, `[]`
	})
	for _, strict := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "config.yaml")
		original := "repository:\n  description: old\ncustom_properties:\n  team: platform\n"
		if err := os.WriteFile(path, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		args := []string{"export", "--repo", "o/r", "--config", path}
		if strict {
			args = append(args, "--strict")
		}
		var out, errout bytes.Buffer
		code := Run(args, strings.NewReader(""), &out, &errout, "test")
		b, _ := os.ReadFile(path)
		if strict {
			if code != 1 || string(b) != original {
				t.Fatalf("code=%d file=%s stderr=%s", code, b, &errout)
			}
		} else if code != 0 || !strings.Contains(string(b), "description: new") || !strings.Contains(string(b), "team: platform") {
			t.Fatalf("code=%d file=%s", code, b)
		}
		if !strings.Contains(errout.String(), "custom_properties") {
			t.Fatal("missing diagnostic")
		}
	}
}
func TestApplyStrictReadFailureAndJSONCompleteness(t *testing.T) {
	writes := 0
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.Method != "GET" {
			writes++
			return 204, ""
		}
		if r.URL.Path == "/repos/o/r" {
			return 200, `{"description":"old"}`
		}
		if strings.HasSuffix(r.URL.Path, "/properties/values") {
			return 403, `{}`
		}
		return 200, `[]`
	})
	path := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(path, []byte("repository:\n  description: new\ncustom_properties: {}\n"), 0600)
	for _, flags := range [][]string{{"--strict", "-y"}, {"--strict", "--dry-run", "--json"}} {
		var out, errout bytes.Buffer
		args := append([]string{"apply", "--repo", "o/r", "--config", path}, flags...)
		if code := Run(args, strings.NewReader(""), &out, &errout, "test"); code != 1 || writes != 0 {
			t.Fatalf("code=%d writes=%d stderr=%s", code, writes, &errout)
		}
		if flags[len(flags)-1] == "--json" {
			var p struct {
				Complete bool
				Skipped  []any
			}
			if err := json.Unmarshal(out.Bytes(), &p); err != nil || p.Complete || len(p.Skipped) != 1 {
				t.Fatalf("json=%s err=%v", &out, err)
			}
		}
	}
}
func TestStrictApplyVerifiesEffectiveValues(t *testing.T) {
	for _, effective := range []bool{false, true} {
		t.Run(map[bool]string{false: "ignored write", true: "effective write"}[effective], func(t *testing.T) {
			description := "old"
			writes := 0
			mockAPI(t, func(r *http.Request) (int, string) {
				if r.Method == "PATCH" {
					writes++
					if effective {
						description = "new"
					}
					return 200, `{}`
				}
				if r.URL.Path == "/repos/o/r" {
					return 200, `{"description":"` + description + `"}`
				}
				return 200, `[]`
			})
			path := filepath.Join(t.TempDir(), "config.yaml")
			os.WriteFile(path, []byte("repository:\n  description: new\n"), 0600)
			var out, errout bytes.Buffer
			code := Run([]string{"apply", "--repo", "o/r", "--config", path, "--strict", "-y"}, strings.NewReader(""), &out, &errout, "test")
			want := 1
			if effective {
				want = 0
			}
			if code != want || writes != 1 {
				t.Fatalf("code=%d writes=%d stderr=%s", code, writes, &errout)
			}
		})
	}
}

func TestDefaultApplyDeniedWriteContinuesAndSucceedsWithWarning(t *testing.T) {
	writes := 0
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.Method == "PATCH" {
			writes++
			if strings.HasSuffix(r.URL.Path, "/properties/values") {
				body, _ := io.ReadAll(r.Body)
				if strings.Contains(string(body), `"a"`) {
					return 403, `{"message":"denied"}`
				}
			}
			return 204, ""
		}
		if r.URL.Path == "/repos/o/r" {
			return 200, `{}`
		}
		return 200, `[]`
	})
	path := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(path, []byte("custom_properties:\n  a: first\n  b: second\n"), 0600)
	var out, errout bytes.Buffer
	code := Run([]string{"apply", "--repo", "o/r", "--config", path, "-y"}, strings.NewReader(""), &out, &errout, "test")
	if code != 0 || writes != 2 || !strings.Contains(errout.String(), "custom_properties.a skipped") || !strings.Contains(out.String(), "Applied 1 change(s); skipped 1") {
		t.Fatalf("code=%d writes=%d stdout=%s stderr=%s", code, writes, &out, &errout)
	}
}

func TestStrictApplyReportsPendingInvitation(t *testing.T) {
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.Method != "GET" {
			t.Fatalf("unexpected mutation: %s", r.URL)
		}
		if r.URL.Path == "/repos/o/r" {
			return 200, `{}`
		}
		if strings.HasSuffix(r.URL.Path, "/invitations") {
			return 200, `[{"id":1,"permissions":"push","invitee":{"login":"alice"}}]`
		}
		return 200, `[]`
	})
	path := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(path, []byte("collaborators:\n  alice:\n    permission: push\n"), 0600)
	var out, errout bytes.Buffer
	code := Run([]string{"apply", "--repo", "o/r", "--config", path, "--strict", "-y"}, strings.NewReader(""), &out, &errout, "test")
	if code != 1 || !strings.Contains(errout.String(), "invitations remain pending: alice") {
		t.Fatalf("code=%d stderr=%s", code, &errout)
	}
}
func TestStrictApplyPollsCodeScanningSetup(t *testing.T) {
	reads, writes := 0, 0
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.Method == "PATCH" {
			writes++
			return 202, `{}`
		}
		if r.URL.Path == "/repos/o/r" {
			return 200, `{}`
		}
		if strings.HasSuffix(r.URL.Path, "/code-scanning/default-setup") {
			reads++
			if reads < 3 {
				return 200, `{"state":"not-configured"}`
			}
			return 200, `{"state":"configured"}`
		}
		return 200, `[]`
	})
	path := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(path, []byte("security:\n  code_scanning_default_setup:\n    state: configured\n"), 0600)
	var out, errout bytes.Buffer
	code := Run([]string{"apply", "--repo", "o/r", "--config", path, "--strict", "-y"}, strings.NewReader(""), &out, &errout, "test")
	if code != 0 || writes != 1 || reads != 3 {
		t.Fatalf("code=%d writes=%d reads=%d stderr=%s", code, writes, reads, &errout)
	}
}
