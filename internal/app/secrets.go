package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

type secretUpload struct {
	Path        string `json:"path"`
	Operation   string `json:"operation"`
	application string
	name        string
	secret      config.EncryptedSecret
}

type secretPlan struct {
	Repository        string               `json:"repository"`
	PreflightComplete bool                 `json:"preflight_complete"`
	ValueComparison   string               `json:"value_comparison"`
	Uploads           []secretUpload       `json:"uploads"`
	Skipped           []github.Unavailable `json:"skipped,omitempty"`
}

func runSecretsApply(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	o, err := parseFlags("secrets apply", args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return reportError(stderr, err)
	}
	owner, repo, _, err := resolveRepo(o.repo)
	if err != nil {
		return reportError(stderr, err)
	}
	path := o.config
	if path == "" {
		path = ".ghrepocfg.yaml"
		if root, ok := gitRoot(); ok {
			path = filepath.Join(root, path)
		}
	}
	desired, err := config.Load(path)
	if err != nil {
		return reportError(stderr, fmt.Errorf("load %s: %w", path, err))
	}
	scopes := desired.SecretScopes()
	count := 0
	for _, scope := range scopes {
		count += len(*scope.Entries)
		for _, name := range sortedSecretNames(*scope.Entries) {
			if !strings.EqualFold((*scope.Entries)[name].Repository, owner+"/"+repo) {
				return reportError(stderr, fmt.Errorf("%s.secrets.%s.repository does not match target %s/%s; encrypt separately for each target and application", scope.Application, name, owner, repo))
			}
		}
	}
	plan := secretPlan{Repository: owner + "/" + repo, PreflightComplete: true, ValueComparison: "unavailable", Uploads: make([]secretUpload, 0)}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var client *github.Client
	if count > 0 {
		token, source, err := resolveToken()
		if err != nil {
			return reportError(stderr, err)
		}
		if o.verbose {
			fmt.Fprintf(stderr, "Authenticated using %s\n", source)
		}
		client = github.NewClient(token)
		for _, scope := range scopes {
			if len(*scope.Entries) == 0 {
				continue
			}
			key, err := client.RepositorySecretPublicKey(ctx, owner, repo, scope.Application)
			if err != nil {
				reason, denied := github.UnavailableReason(err, true)
				if !denied {
					return reportError(stderr, fmt.Errorf("%s.secrets: %w", scope.Application, actionableAPIError(err)))
				}
				plan.PreflightComplete = false
				for _, name := range sortedSecretNames(*scope.Entries) {
					plan.Skipped = append(plan.Skipped, github.Unavailable{Path: scope.Application + ".secrets." + name, Reason: reason})
				}
				continue
			}
			for _, name := range sortedSecretNames(*scope.Entries) {
				secret := (*scope.Entries)[name]
				path := scope.Application + ".secrets." + name
				if secret.KeyID != key.KeyID {
					return reportError(stderr, fmt.Errorf("%s.key_id does not match the target application's current repository key; re-encrypt the secret", path))
				}
				plan.Uploads = append(plan.Uploads, secretUpload{Path: path, Operation: "upload", application: scope.Application, name: name, secret: secret})
			}
		}
	}
	printUnavailable(stderr, "apply", plan.Skipped, false)
	if o.json {
		writeJSON(stdout, plan)
	} else {
		fmt.Fprintf(stdout, "Repository: %s\n\n", plan.Repository)
		if count == 0 {
			fmt.Fprintln(stdout, "No secret uploads configured. Existing secrets are untouched.")
		} else {
			fmt.Fprintln(stdout, "Secret values cannot be compared or verified. Every listed payload will be uploaded.")
			for _, upload := range plan.Uploads {
				fmt.Fprintf(stdout, "\n  %s\n    upload configured encrypted value\n", upload.Path)
			}
		}
	}
	if o.strict && count > 0 {
		return reportError(stderr, errors.New("strict apply cannot verify write-only secret values; no secrets uploaded"))
	}
	if o.dryRun {
		if len(plan.Uploads) > 0 {
			return exitDrift
		}
		return exitOK
	}
	if len(plan.Uploads) == 0 {
		return exitOK
	}
	if !o.yes && !confirm(stdin, stderr, len(plan.Uploads)) {
		return reportError(stderr, errors.New("secret upload cancelled"))
	}
	applied, skipped := 0, len(plan.Skipped)
	for _, upload := range plan.Uploads {
		path := upload.Path
		err := client.SetRepositorySecret(ctx, owner, repo, upload.application, upload.name, upload.secret)
		if err != nil {
			// Public-key reads can also fail after the confirmation prompt.
			var apiErr *github.APIError
			reading := errors.As(err, &apiErr) && apiErr.Method == "GET"
			if reason, denied := github.UnavailableReason(err, reading); denied {
				printUnavailable(stderr, "apply", []github.Unavailable{{Path: path, Reason: reason}}, false)
				skipped++
				continue
			}
			return reportError(stderr, fmt.Errorf("%s: %w (earlier uploads are retained)", path, actionableAPIError(err)))
		}
		applied++
		fmt.Fprintf(stdout, "Uploaded %s (accepted by GitHub; value not verified)\n", path)
	}
	fmt.Fprintf(stdout, "Uploaded %d secret(s); skipped %d.\n", applied, skipped)
	return exitOK
}

func sortedSecretNames(entries map[string]config.EncryptedSecret) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
