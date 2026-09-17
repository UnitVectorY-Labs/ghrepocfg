package app

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/reconcile"
)

func noChangesMessage(complete bool) string {
	if !complete {
		return "No changes to readable attributes; evaluation is incomplete."
	}
	return "No changes."
}
func printUnavailable(w io.Writer, operation string, unavailable []github.Unavailable, preserve bool) {
	s := styleFor(w)
	for _, u := range unavailable {
		action := "skipped"
		if operation == "export" {
			action = "omitted from export"
			if preserve {
				action = "not refreshed (existing value retained)"
			}
		}
		fmt.Fprintf(w, "%s %s %s: %s.\n", s.yellow(s.bold("warning:")), u.Path, action, strings.TrimSuffix(u.Reason, "."))
	}
}

func verifyManaged(ctx context.Context, client *github.Client, owner, repo string, desired *config.Config, verbose bool) error {
	// Give verification, including asynchronous settings, a bounded deadline.
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	for {
		state, err := client.Read(ctx, owner, repo, scopeFor(desired, verbose))
		if err != nil {
			return fmt.Errorf("verify applied state: %w", err)
		}
		plan := reconcile.Build(owner, repo, desired, state, client, false)
		if !plan.Complete {
			return fmt.Errorf("strict verification incomplete: %v", plan.Skipped)
		}
		if desired.Collaborators != nil {
			var names []string
			for name := range *desired.Collaborators {
				names = append(names, name)
			}
			if pending := reconcile.PendingInvitations(names, state); len(pending) > 0 {
				return fmt.Errorf("strict verification: collaborator invitations remain pending: %s", strings.Join(pending, ", "))
			}
		}
		if !plan.Drift {
			return nil
		}
		paths := make([]string, 0, len(plan.Changes))
		async := true
		for _, change := range plan.Changes {
			paths = append(paths, change.Path)
			if change.Path != "security.code_scanning_default_setup" {
				async = false
			}
		}
		if !async {
			return fmt.Errorf("strict verification: requested values are not effective: %s", strings.Join(paths, ", "))
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
