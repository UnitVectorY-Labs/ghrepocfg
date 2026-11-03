package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func secretFixture(t *testing.T, names ...string) (string, string) {
	t.Helper()
	ciphertext := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{42}, 64))
	var yaml strings.Builder
	yaml.WriteString("actions:\n  secrets:\n")
	for _, name := range names {
		yaml.WriteString("    " + name + ":\n      repository: o/r\n      key_id: '123'\n      encrypted_value: " + ciphertext + "\n")
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml.String()), 0600); err != nil {
		t.Fatal(err)
	}
	return path, ciphertext
}

func keyResponse(id string) string {
	return `{"key_id":"` + id + `","key":"` + base64.StdEncoding.EncodeToString(make([]byte, 32)) + `"}`
}

func TestSecretsApplyUploadsExactCiphertextEveryTime(t *testing.T) {
	path, ciphertext := secretFixture(t, "TOKEN")
	writes, reads := 0, 0
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/secrets/public-key") {
			reads++
			return 200, keyResponse("123")
		}
		if r.Method != "PUT" || r.URL.Path != "/repos/o/r/actions/secrets/TOKEN" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if len(payload) != 2 || payload["key_id"] != "123" || payload["encrypted_value"] != ciphertext {
			t.Fatalf("unexpected payload keys or ciphertext")
		}
		writes++
		return 204, ""
	})
	for i := 0; i < 2; i++ {
		var out, errout bytes.Buffer
		code := Run([]string{"secrets", "apply", "--repo", "o/r", "--config", path, "-y"}, strings.NewReader(""), &out, &errout, "test")
		if code != 0 || !strings.Contains(out.String(), "Uploaded 1 secret(s)") || strings.Contains(out.String()+errout.String(), ciphertext) {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &errout)
		}
	}
	if writes != 2 || reads != 4 {
		t.Fatalf("writes=%d reads=%d", writes, reads)
	}
}

func TestSecretsApplyPreflightAndNonMutatingModes(t *testing.T) {
	for _, tc := range []struct {
		name, target, key, input string
		flags                    []string
		want                     int
	}{
		{"dry run", "o/r", "123", "", []string{"--dry-run"}, 2},
		{"JSON dry run", "o/r", "123", "", []string{"--dry-run", "--json"}, 2},
		{"strict", "o/r", "123", "", []string{"--strict", "-y"}, 1},
		{"strict JSON", "o/r", "123", "", []string{"--strict", "--dry-run", "--json"}, 1},
		{"wrong target", "o/other", "123", "", []string{"-y"}, 1},
		{"stale key", "o/r", "456", "", []string{"-y"}, 1},
		{"cancel", "o/r", "123", "n\n", nil, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, ciphertext := secretFixture(t, "TOKEN")
			mockAPI(t, func(r *http.Request) (int, string) {
				if r.Method != "GET" {
					t.Fatal("unexpected mutation")
				}
				return 200, keyResponse(tc.key)
			})
			args := append([]string{"secrets", "apply", "--repo", tc.target, "--config", path}, tc.flags...)
			var out, errout bytes.Buffer
			code := Run(args, strings.NewReader(tc.input), &out, &errout, "test")
			if code != tc.want || strings.Contains(out.String()+errout.String(), ciphertext) {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &errout)
			}
			if strings.Contains(tc.name, "JSON") {
				var p secretPlan
				if err := json.Unmarshal(out.Bytes(), &p); err != nil || p.ValueComparison != "unavailable" || len(p.Uploads) != 1 {
					t.Fatalf("plan=%s err=%v", &out, err)
				}
			}
		})
	}
}

func TestSecretsApplyPermissionDenialsAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		status, want, writes int
		keyRead              bool
	}{
		{"denied upload continues", 403, 0, 2, false},
		{"invalid upload fails", 422, 1, 1, false},
		{"rate limit fails", 429, 1, 1, false},
		{"denied key skips all", 403, 0, 0, true},
		{"hidden key skips all", 404, 0, 0, true},
		{"server failure fails", 500, 1, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, _ := secretFixture(t, "A", "B")
			writes := 0
			mockAPI(t, func(r *http.Request) (int, string) {
				if r.Method == "GET" {
					if tc.keyRead {
						return tc.status, `{}`
					}
					return 200, keyResponse("123")
				}
				writes++
				if strings.HasSuffix(r.URL.Path, "/A") {
					return tc.status, `{}`
				}
				return 204, ""
			})
			var out, errout bytes.Buffer
			code := Run([]string{"secrets", "apply", "--repo", "o/r", "--config", path, "-y"}, strings.NewReader(""), &out, &errout, "test")
			if code != tc.want || writes != tc.writes {
				t.Fatalf("code=%d writes=%d stderr=%s", code, writes, &errout)
			}
			if tc.want == 0 && !strings.Contains(errout.String(), "actions.secrets.A skipped") {
				t.Fatalf("missing skip warning: %s", &errout)
			}
		})
	}
}

func TestSecretsRechecksKeyAfterConfirmation(t *testing.T) {
	path, _ := secretFixture(t, "TOKEN")
	reads := 0
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.Method != "GET" {
			t.Fatal("must not upload with changed key")
		}
		reads++
		if reads == 1 {
			return 200, keyResponse("123")
		}
		return 200, keyResponse("456")
	})
	var out, errout bytes.Buffer
	if code := Run([]string{"secrets", "apply", "--repo", "o/r", "--config", path, "-y"}, strings.NewReader(""), &out, &errout, "test"); code != 1 || !strings.Contains(errout.String(), "key changed") {
		t.Fatalf("code=%d stderr=%s", code, &errout)
	}
}

func TestOrdinaryApplySkipsSecretsAndContinuesIndependentWork(t *testing.T) {
	for _, strict := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "strict"}[strict], func(t *testing.T) {
			path, _ := secretFixture(t, "TOKEN")
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			io.WriteString(f, "repository:\n  description: new\n")
			f.Close()
			writes := 0
			mockAPI(t, func(r *http.Request) (int, string) {
				if r.Method == "PATCH" {
					writes++
					return 200, `{}`
				}
				if r.Method == "GET" && (strings.HasSuffix(r.URL.Path, "/branches") || strings.HasSuffix(r.URL.Path, "/tags/protection")) {
					return 200, `[]`
				}
				if r.Method != "GET" || r.URL.Path != "/repos/o/r" {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL)
				}
				return 200, `{"description":"old"}`
			})
			args := []string{"apply", "--repo", "o/r", "--config", path, "-y"}
			wantCode, wantWrites := 0, 1
			if strict {
				args = append(args, "--strict")
				wantCode, wantWrites = 1, 0
			}
			var out, errout bytes.Buffer
			code := Run(args, strings.NewReader(""), &out, &errout, "test")
			if code != wantCode || writes != wantWrites || !strings.Contains(errout.String(), "ghrepocfg secrets apply") {
				t.Fatalf("code=%d writes=%d stderr=%s", code, writes, &errout)
			}
		})
	}
}

func TestExportRetainsCiphertextAndStrictDoesNotWrite(t *testing.T) {
	for _, full := range []bool{false, true} {
		for _, strict := range []bool{false, true} {
			t.Run(map[bool]string{false: "scoped", true: "full"}[full]+map[bool]string{false: " default", true: " strict"}[strict], func(t *testing.T) {
				path, ciphertexts := allApplicationsFixture(t)
				before, _ := os.ReadFile(path)
				mockAPI(t, func(r *http.Request) (int, string) {
					if r.Method != "GET" {
						t.Fatal("unexpected mutation")
					}
					if r.URL.Path == "/repos/o/r" {
						return 200, `{"description":"current"}`
					}
					// Keep full discovery bounded without inventing empty state.
					return 403, `{}`
				})
				args := []string{"export", "--repo", "o/r", "--config", path}
				if full {
					args = append(args, "--full")
				}
				if strict {
					args = append(args, "--strict")
				}
				var out, errout bytes.Buffer
				code := Run(args, strings.NewReader(""), &out, &errout, "test")
				after, _ := os.ReadFile(path)
				for application, value := range ciphertexts {
					if !strings.Contains(string(after), value) || !strings.Contains(errout.String(), application+".secrets") {
						t.Fatalf("export lost %s ciphertext or diagnostic", application)
					}
				}
				want := 0
				if strict {
					want = 1
					if !bytes.Equal(before, after) {
						t.Fatal("strict export modified file")
					}
				}
				if code != want {
					t.Fatalf("code=%d stderr=%s", code, &errout)
				}
			})
		}
	}
}

func TestEmptySecretsNeverReadOrDelete(t *testing.T) {
	for _, yaml := range []string{"actions:\n  secrets: {}\nagents:\n  secrets: {}\ncodespaces:\n  secrets: {}\ndependabot:\n  secrets: {}\n", "repository:\n  description: unrelated\n"} {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(yaml), 0600); err != nil {
			t.Fatal(err)
		}
		mockAPI(t, func(r *http.Request) (int, string) {
			t.Fatalf("empty secret scope must not call GitHub: %s %s", r.Method, r.URL)
			return 500, ""
		})
		var out, errout bytes.Buffer
		code := Run([]string{"secrets", "apply", "--repo", "o/r", "--config", path, "--strict", "-y"}, strings.NewReader(""), &out, &errout, "test")
		if code != 0 || !strings.Contains(out.String(), "No secret uploads configured") {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &errout)
		}
	}
}

func TestInvalidSecretPublicKeyNeverUploads(t *testing.T) {
	for _, body := range []string{`{}`, `{"key_id":"123","key":"not-base64"}`, `{"key_id":"123","key":"aGk="}`} {
		path, _ := secretFixture(t, "TOKEN")
		mockAPI(t, func(r *http.Request) (int, string) {
			if r.Method != "GET" {
				t.Fatal("invalid public key must prevent upload")
			}
			return 200, body
		})
		var out, errout bytes.Buffer
		code := Run([]string{"secrets", "apply", "--repo", "o/r", "--config", path, "-y"}, strings.NewReader(""), &out, &errout, "test")
		if code != 1 || !strings.Contains(errout.String(), "public key is missing or invalid") {
			t.Fatalf("code=%d stderr=%s", code, &errout)
		}
	}
}

func allApplicationsFixture(t *testing.T) (string, map[string]string) {
	t.Helper()
	var yaml strings.Builder
	ciphertexts := map[string]string{}
	for i, application := range []string{"actions", "agents", "codespaces", "dependabot"} {
		ciphertext := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{byte(i + 1)}, 64))
		ciphertexts[application] = ciphertext
		fmt.Fprintf(&yaml, "%s:\n  secrets:\n    TOKEN:\n      repository: o/r\n      key_id: '%s-key'\n      encrypted_value: %s\n", application, application, ciphertext)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml.String()), 0600); err != nil {
		t.Fatal(err)
	}
	return path, ciphertexts
}

func TestSecretsApplicationsUseSeparateKeysAndPayloads(t *testing.T) {
	path, ciphertexts := allApplicationsFixture(t)
	writes := map[string]int{}
	reads := map[string]int{}
	mockAPI(t, func(r *http.Request) (int, string) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 6 || parts[0] != "repos" || parts[4] != "secrets" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		application := parts[3]
		ciphertext, ok := ciphertexts[application]
		if !ok {
			t.Fatalf("wrong application %s", application)
		}
		if r.Method == "GET" && parts[5] == "public-key" {
			reads[application]++
			return 200, keyResponse(application + "-key")
		}
		if r.Method != "PUT" || parts[5] != "TOKEN" {
			t.Fatalf("unexpected mutation %s %s", r.Method, r.URL)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 2 || body["key_id"] != application+"-key" || body["encrypted_value"] != ciphertext {
			t.Fatalf("payload routed to wrong application %s", application)
		}
		writes[application]++
		return 201, ""
	})
	for _, flags := range [][]string{{"--dry-run", "--json"}, {"--yes"}, {"--yes"}} {
		var out, errout bytes.Buffer
		code := Run(append([]string{"secrets", "apply", "--repo", "o/r", "--config", path}, flags...), strings.NewReader(""), &out, &errout, "test")
		want := 0
		if flags[0] == "--dry-run" {
			want = 2
		}
		if code != want {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &errout)
		}
		for app, ciphertext := range ciphertexts {
			if strings.Contains(out.String()+errout.String(), ciphertext) {
				t.Fatal("ciphertext exposed in output")
			}
			if !strings.Contains(out.String(), app+".secrets.TOKEN") {
				t.Fatalf("missing application path %s", app)
			}
		}
	}
	for application := range ciphertexts {
		if writes[application] != 2 || reads[application] != 5 {
			t.Fatalf("%s writes=%d reads=%d", application, writes[application], reads[application])
		}
	}
}

func TestSecretApplicationDenialsContinueOtherApplications(t *testing.T) {
	for _, deniedApp := range []string{"actions", "agents", "codespaces", "dependabot"} {
		for _, deniedMethod := range []string{"GET", "PUT"} {
			t.Run(deniedApp+"/"+deniedMethod, func(t *testing.T) {
				path, _ := allApplicationsFixture(t)
				uploaded := map[string]bool{}
				mockAPI(t, func(r *http.Request) (int, string) {
					application := strings.Split(r.URL.Path, "/")[4]
					if application == deniedApp && r.Method == deniedMethod {
						return 403, `{"message":"denied"}`
					}
					if r.Method == "GET" {
						return 200, keyResponse(application + "-key")
					}
					uploaded[application] = true
					return 204, ""
				})
				var out, errout bytes.Buffer
				code := Run([]string{"secrets", "apply", "--repo", "o/r", "--config", path, "--yes"}, strings.NewReader(""), &out, &errout, "test")
				if code != 0 || len(uploaded) != 3 || uploaded[deniedApp] || !strings.Contains(errout.String(), deniedApp+".secrets.TOKEN skipped") {
					t.Fatalf("code=%d uploads=%v stderr=%s", code, uploaded, &errout)
				}
			})
		}
	}
}

func TestSecretKeyFromAnotherApplicationRejectedBeforeAnyUpload(t *testing.T) {
	path, _ := allApplicationsFixture(t)
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.Method != "GET" {
			t.Fatal("must reject all writes before confirmation")
		}
		application := strings.Split(r.URL.Path, "/")[4]
		if application == "codespaces" {
			return 200, keyResponse("actions-key")
		}
		return 200, keyResponse(application + "-key")
	})
	var out, errout bytes.Buffer
	code := Run([]string{"secrets", "apply", "--repo", "o/r", "--config", path, "--yes"}, strings.NewReader(""), &out, &errout, "test")
	if code != 1 || !strings.Contains(errout.String(), "codespaces.secrets.TOKEN.key_id") {
		t.Fatalf("code=%d stderr=%s", code, &errout)
	}
}

func TestOrdinaryApplyRecognizesAllSecretScopes(t *testing.T) {
	path, _ := allApplicationsFixture(t)
	mockAPI(t, func(r *http.Request) (int, string) {
		if r.Method != "GET" {
			t.Fatal("ordinary apply must never upload secrets")
		}
		if r.URL.Path == "/repos/o/r" {
			return 200, `{}`
		}
		return 200, `[]`
	})
	for _, command := range [][]string{{"apply"}, {"apply", "--strict"}, {"secrets", "apply", "--strict"}} {
		var out, errout bytes.Buffer
		args := append(append([]string{}, command...), "--repo", "o/r", "--config", path, "--yes")
		code := Run(args, strings.NewReader(""), &out, &errout, "test")
		want := 1
		if len(command) == 1 {
			want = 0
		}
		if code != want {
			t.Fatalf("code=%d stderr=%s", code, &errout)
		}
		if command[0] == "apply" {
			for _, application := range []string{"actions", "agents", "codespaces", "dependabot"} {
				if !strings.Contains(errout.String(), application+".secrets skipped") {
					t.Fatalf("missing skip warning for %s: %s", application, &errout)
				}
			}
		}
	}
}
