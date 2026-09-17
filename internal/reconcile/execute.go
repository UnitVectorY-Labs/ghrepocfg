package reconcile

import (
	"context"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

type Result struct {
	Applied []string
	Skipped []github.Unavailable
	Failed  []Failure
}

// ExecuteWithPolicy continues independent operations after an access denial.
// Strict execution stops on the first failed or skipped mutation.
func (p *Plan) ExecuteWithPolicy(ctx context.Context, strict bool) Result {
	r := Result{Skipped: append([]github.Unavailable(nil), p.Skipped...)}
	if strict && len(r.Skipped) > 0 {
		return r
	}
	blocked := map[string]bool{}
	for index, c := range p.Changes {
		dependency := p.dependencies[c.Path]
		if c.Path == "actions.selected_actions" {
			dependency = "actions.permissions"
		}

		if blocked[dependency] {
			r.Skipped = append(r.Skipped, github.Unavailable{Path: c.Path, Reason: "required operation " + dependency + " did not complete"})
			continue
		}
		err := c.apply(ctx)
		if err == nil {
			r.Applied = append(r.Applied, c.Path)
			continue
		}
		blocked[c.Path] = true
		if reason, ok := github.UnavailableReason(err, false); ok {
			r.Skipped = append(r.Skipped, github.Unavailable{Path: c.Path, Reason: reason})
		} else {
			r.Failed = append(r.Failed, Failure{Path: c.Path, Error: err.Error()})
		}
		if strict || ctx.Err() != nil || github.IsRateLimit(err) {
			for _, remaining := range p.Changes[index+1:] {
				r.Skipped = append(r.Skipped, github.Unavailable{Path: remaining.Path, Reason: "not attempted after execution stopped at " + c.Path})
			}
			break
		}
	}
	return r
}

// Pending invitations are effective invitations, not accepted access grants.
func PendingInvitations(desiredNames []string, state *github.State) []string {
	var pending []string
	for _, name := range desiredNames {
		if c, ok := state.Collaborators[strings.ToLower(name)]; ok && c.InvitationID != nil {
			pending = append(pending, name)
		}
	}
	return pending
}
