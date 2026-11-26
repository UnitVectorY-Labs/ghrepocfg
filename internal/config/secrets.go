package config

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// EncryptedSecret is a write-only repository secret deployment artifact.
// The containing section binds the ciphertext to its application.
// Entries are upserts: omitted names and an empty map never delete secrets.
type EncryptedSecret struct {
	Repository     string `yaml:"repository" json:"repository"`
	KeyID          string `yaml:"key_id" json:"key_id"`
	EncryptedValue string `yaml:"encrypted_value" json:"encrypted_value"`
}

// SecretSettings holds repository secret uploads for an application.
type SecretSettings struct {
	Secrets *map[string]EncryptedSecret `yaml:"secrets,omitempty" json:"secrets,omitzero"`
}

// SecretScope identifies an application's configured uploads, including empty maps.
type SecretScope struct {
	Application string
	Entries     *map[string]EncryptedSecret
}

func ValidSecretApplication(application string) bool {
	switch application {
	case "actions", "agents", "codespaces", "dependabot":
		return true
	default:
		return false
	}
}

// SecretScopes returns configured maps in a stable application order.
func (c *Config) SecretScopes() []SecretScope {
	if c == nil {
		return nil
	}
	var scopes []SecretScope
	if c.Actions != nil && c.Actions.Secrets != nil {
		scopes = append(scopes, SecretScope{"actions", c.Actions.Secrets})
	}
	for _, item := range []struct {
		application string
		settings    *SecretSettings
	}{
		{"agents", c.Agents}, {"codespaces", c.Codespaces}, {"dependabot", c.Dependabot},
	} {
		if item.settings != nil && item.settings.Secrets != nil {
			scopes = append(scopes, SecretScope{item.application, item.settings.Secrets})
		}
	}
	return scopes
}

// PreserveSecrets retains deployment artifacts that cannot be read back from GitHub.
func (c *Config) PreserveSecrets(base *Config) {
	for _, scope := range base.SecretScopes() {
		switch scope.Application {
		case "actions":
			if c.Actions == nil {
				c.Actions = &ActionsSettings{}
			}
			c.Actions.Secrets = scope.Entries
		case "agents":
			c.Agents = &SecretSettings{Secrets: scope.Entries}
		case "codespaces":
			c.Codespaces = &SecretSettings{Secrets: scope.Entries}
		case "dependabot":
			c.Dependabot = &SecretSettings{Secrets: scope.Entries}
		}
	}
}

func (c *Config) validateSecrets() error {
	for _, scope := range c.SecretScopes() {
		if err := validateSecretEntries(scope.Application+".secrets", *scope.Entries); err != nil {
			return err
		}
	}
	return nil
}

func validateSecretEntries(prefix string, entries map[string]EncryptedSecret) error {
	seen := map[string]bool{}
	for name, secret := range entries {
		upper := strings.ToUpper(name)
		path := prefix + "." + name
		if !variableNamePattern.MatchString(name) || strings.HasPrefix(upper, "GITHUB_") || seen[upper] {
			return fmt.Errorf("invalid configuration: %s has an invalid or duplicate secret name", path)
		}
		seen[upper] = true
		parts := strings.Split(secret.Repository, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(secret.Repository, " \t\r\n?#%\\:") || parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
			return fmt.Errorf("invalid configuration: %s.repository must be OWNER/REPO", path)
		}
		if secret.KeyID == "" || strings.TrimSpace(secret.KeyID) != secret.KeyID {
			return fmt.Errorf("invalid configuration: %s.key_id is required", path)
		}
		ciphertext, err := base64.StdEncoding.Strict().DecodeString(secret.EncryptedValue)
		// A libsodium sealed box adds 48 bytes; GitHub secrets allow 48 KiB plaintext.
		if err != nil || len(ciphertext) < 48 || len(ciphertext) > 48*1024+48 || strings.ContainsAny(secret.EncryptedValue, "\r\n") {
			return fmt.Errorf("invalid configuration: %s.encrypted_value must be a base64-encoded sealed box (48 through 49200 decoded bytes)", path)
		}
	}
	return nil
}
