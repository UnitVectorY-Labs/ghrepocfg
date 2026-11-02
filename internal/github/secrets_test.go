package github

import (
	"context"
	"net/http"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

func TestSecretApplicationValidationBeforeRequests(t *testing.T) {
	client := testClient(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("invalid application made a request: %s", r.URL)
		return nil, nil
	})
	for _, application := range []string{"", "unknown", "../actions", "Actions"} {
		if _, err := client.RepositorySecretPublicKey(context.Background(), "o", "r", application); err == nil {
			t.Fatalf("accepted invalid application %q", application)
		}
		if err := client.SetRepositorySecret(context.Background(), "o", "r", application, "TOKEN", config.EncryptedSecret{}); err == nil {
			t.Fatalf("accepted invalid application %q", application)
		}
	}
}

func TestSecretPermissionHints(t *testing.T) {
	for _, tc := range []struct{ application, read, write string }{
		{"actions", "Secrets: read", "Secrets: write"},
		{"agents", "Agent secrets: read", "Agent secrets: write"},
		{"codespaces", "Codespaces secrets: write", "Codespaces secrets: write"},
		{"dependabot", "Dependabot secrets: read", "Dependabot secrets: write"},
	} {
		path := "/repos/o/r/" + tc.application + "/secrets/"
		if got := RequiredPermission("GET", path+"public-key"); got != tc.read {
			t.Fatalf("%s GET hint=%s", tc.application, got)
		}
		if got := RequiredPermission("PUT", path+"TOKEN"); got != tc.write {
			t.Fatalf("%s PUT hint=%s", tc.application, got)
		}
	}
}
