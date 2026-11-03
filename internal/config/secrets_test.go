package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncryptedSecretsValidation(t *testing.T) {
	ciphertext := base64.StdEncoding.EncodeToString(make([]byte, 64))
	valid := "actions:\n  secrets:\n    TOKEN:\n      repository: o/r\n      key_id: \"123\"\n      encrypted_value: " + ciphertext + "\n"
	for _, application := range []string{"actions", "agents", "codespaces", "dependabot"} {
		for _, tc := range []struct {
			name, text string
			valid      bool
		}{
			{"valid", valid, true},
			{"empty map", "actions:\n  secrets: {}\n", true},
			{"bad base64", strings.ReplaceAll(valid, ciphertext, "plaintext"), false},
			{"short box", strings.ReplaceAll(valid, ciphertext, "aGk="), false},
			{"too large", strings.ReplaceAll(valid, ciphertext, base64.StdEncoding.EncodeToString(make([]byte, 49201))), false},
			{"missing target", strings.ReplaceAll(valid, "repository: o/r", "repository: ''"), false},
			{"target URL", strings.ReplaceAll(valid, "repository: o/r", "repository: https://github.com/o/r"), false},
			{"missing key", strings.ReplaceAll(valid, "key_id: \"123\"", "key_id: ''"), false},
			{"reserved name", strings.ReplaceAll(valid, "TOKEN:", "GITHUB_TOKEN:"), false},
			{"invalid name", strings.ReplaceAll(valid, "TOKEN:", "1TOKEN:"), false},
			{"duplicate case", valid + "    token:\n      repository: o/r\n      key_id: '123'\n      encrypted_value: " + ciphertext, false},
			{"plaintext field", valid + "      value: plaintext\n", false},
		} {
			t.Run(application+"/"+tc.name, func(t *testing.T) {
				c, err := Parse([]byte(strings.Replace(tc.text, "actions:", application+":", 1)))
				if (err == nil) != tc.valid {
					t.Fatalf("valid=%v error=%v", tc.valid, err)
				}
				if tc.valid {
					b, err := Marshal(c)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := Parse(b); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
