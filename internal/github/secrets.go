package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

type SecretPublicKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"`
}

func (c *Client) RepositorySecretPublicKey(ctx context.Context, owner, repo, application string) (SecretPublicKey, error) {
	var key SecretPublicKey
	if !config.ValidSecretApplication(application) {
		return key, fmt.Errorf("unsupported secret application %q", application)
	}
	_, err := c.request(ctx, http.MethodGet, repoPath(owner, repo, "/"+application+"/secrets/public-key"), nil, &key)
	if err != nil {
		return key, err
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(key.Key)
	if err != nil || len(decoded) != 32 || key.KeyID == "" {
		return key, fmt.Errorf("repository secret public key is missing or invalid")
	}
	return key, nil
}

// SetRepositorySecret rechecks the public key after confirmation, then uploads
// already-encrypted data. A successful response is acceptance, not read-back verification.
func (c *Client) SetRepositorySecret(ctx context.Context, owner, repo, application, name string, secret config.EncryptedSecret) error {
	key, err := c.RepositorySecretPublicKey(ctx, owner, repo, application)
	if err != nil {
		return err
	}
	if secret.KeyID != key.KeyID {
		return fmt.Errorf("encryption key changed; re-encrypt the secret with the target repository's current key")
	}
	body := struct {
		KeyID          string `json:"key_id"`
		EncryptedValue string `json:"encrypted_value"`
	}{secret.KeyID, secret.EncryptedValue}
	_, err = c.request(ctx, http.MethodPut, repoPath(owner, repo, "/"+application+"/secrets/"+url.PathEscape(name)), body, nil)
	return err
}
