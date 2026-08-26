// Package exportconfig maps GitHub state to full or scope-preserving YAML.
package exportconfig

import (
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
)

func FromState(s *github.State) *config.Config {
	c := &config.Config{Repository: s.Repository, Security: s.Security, Actions: s.Actions}
	collabs := map[string]config.Access{}
	for name, v := range s.Collaborators {
		collabs[name] = config.Access{Permission: v.Permission}
	}
	c.Collaborators = &collabs
	teams := map[string]config.Access{}
	for name, v := range s.Teams {
		teams[name] = config.Access{Permission: v.Permission}
	}
	c.Teams = &teams
	rules := map[string]config.Ruleset{}
	for name, v := range s.Rulesets {
		rules[name] = v.Value
	}
	c.Rulesets = &rules
	return c
}

func ScopedFromState(base *config.Config, s *github.State) *config.Config {
	out := &config.Config{}
	if base.Repository != nil {
		out.Repository = &config.RepositorySettings{}
		copyPresent(base.Repository, s.Repository, out.Repository)
	}
	if base.Security != nil {
		out.Security = &config.SecuritySettings{}
		if base.Security.VulnerabilityAlerts != nil {
			out.Security.VulnerabilityAlerts = s.Security.VulnerabilityAlerts
		}
		if base.Security.AutomatedSecurityFixes != nil {
			out.Security.AutomatedSecurityFixes = s.Security.AutomatedSecurityFixes
		}
		if base.Security.AdvancedSecurity != nil {
			out.Security.AdvancedSecurity = s.Security.AdvancedSecurity
		}
		if base.Security.CodeSecurity != nil {
			out.Security.CodeSecurity = s.Security.CodeSecurity
		}
		if base.Security.SecretScanning != nil {
			out.Security.SecretScanning = s.Security.SecretScanning
		}
		if base.Security.SecretScanningPushProtection != nil {
			out.Security.SecretScanningPushProtection = s.Security.SecretScanningPushProtection
		}
		if base.Security.SecretScanningAIDetection != nil {
			out.Security.SecretScanningAIDetection = s.Security.SecretScanningAIDetection
		}
		if base.Security.SecretScanningNonProviderPatterns != nil {
			out.Security.SecretScanningNonProviderPatterns = s.Security.SecretScanningNonProviderPatterns
		}
		if base.Security.SecretScanningDelegatedAlertDismissal != nil {
			out.Security.SecretScanningDelegatedAlertDismissal = s.Security.SecretScanningDelegatedAlertDismissal
		}
		if base.Security.SecretScanningDelegatedBypass != nil {
			out.Security.SecretScanningDelegatedBypass = s.Security.SecretScanningDelegatedBypass
		}
		if base.Security.SecretScanningDelegatedBypassOptions != nil {
			out.Security.SecretScanningDelegatedBypassOptions = s.Security.SecretScanningDelegatedBypassOptions
		}
	}
	if base.Actions != nil {
		out.Actions = &config.ActionsSettings{}
		if base.Actions.Enabled != nil {
			out.Actions.Enabled = s.Actions.Enabled
		}
		if base.Actions.AllowedActions != nil {
			out.Actions.AllowedActions = s.Actions.AllowedActions
		}
		if base.Actions.SelectedActions != nil {
			out.Actions.SelectedActions = s.Actions.SelectedActions
		}
		if base.Actions.DefaultWorkflowPermissions != nil {
			out.Actions.DefaultWorkflowPermissions = s.Actions.DefaultWorkflowPermissions
		}
		if base.Actions.CanApprovePullRequestReviews != nil {
			out.Actions.CanApprovePullRequestReviews = s.Actions.CanApprovePullRequestReviews
		}
	}
	if base.Collaborators != nil {
		m := map[string]config.Access{}
		for n, v := range s.Collaborators {
			m[n] = config.Access{Permission: v.Permission}
		}
		out.Collaborators = &m
	}
	if base.Teams != nil {
		m := map[string]config.Access{}
		for n, v := range s.Teams {
			m[n] = config.Access{Permission: v.Permission}
		}
		out.Teams = &m
	}
	if base.Rulesets != nil {
		m := map[string]config.Ruleset{}
		for n, v := range s.Rulesets {
			m[n] = v.Value
		}
		out.Rulesets = &m
	}
	return out
}

func copyPresent(base, cur, out *config.RepositorySettings) {
	if base.Description != nil {
		out.Description = cur.Description
	}
	if base.Homepage != nil {
		out.Homepage = cur.Homepage
	}
	if base.HasIssues != nil {
		out.HasIssues = cur.HasIssues
	}
	if base.HasProjects != nil {
		out.HasProjects = cur.HasProjects
	}
	if base.HasWiki != nil {
		out.HasWiki = cur.HasWiki
	}
	if base.HasDiscussions != nil {
		out.HasDiscussions = cur.HasDiscussions
	}
	if base.HasPullRequests != nil {
		out.HasPullRequests = cur.HasPullRequests
	}
	if base.PullRequestCreationPolicy != nil {
		out.PullRequestCreationPolicy = cur.PullRequestCreationPolicy
	}
	if base.IsTemplate != nil {
		out.IsTemplate = cur.IsTemplate
	}
	if base.DefaultBranch != nil {
		out.DefaultBranch = cur.DefaultBranch
	}
	if base.AllowSquashMerge != nil {
		out.AllowSquashMerge = cur.AllowSquashMerge
	}
	if base.AllowMergeCommit != nil {
		out.AllowMergeCommit = cur.AllowMergeCommit
	}
	if base.AllowRebaseMerge != nil {
		out.AllowRebaseMerge = cur.AllowRebaseMerge
	}
	if base.AllowAutoMerge != nil {
		out.AllowAutoMerge = cur.AllowAutoMerge
	}
	if base.DeleteBranchOnMerge != nil {
		out.DeleteBranchOnMerge = cur.DeleteBranchOnMerge
	}
	if base.AllowUpdateBranch != nil {
		out.AllowUpdateBranch = cur.AllowUpdateBranch
	}
	if base.UseSquashPRTitleAsDefault != nil {
		out.UseSquashPRTitleAsDefault = cur.UseSquashPRTitleAsDefault
	}
	if base.SquashMergeCommitTitle != nil {
		out.SquashMergeCommitTitle = cur.SquashMergeCommitTitle
	}
	if base.SquashMergeCommitMessage != nil {
		out.SquashMergeCommitMessage = cur.SquashMergeCommitMessage
	}
	if base.MergeCommitTitle != nil {
		out.MergeCommitTitle = cur.MergeCommitTitle
	}
	if base.MergeCommitMessage != nil {
		out.MergeCommitMessage = cur.MergeCommitMessage
	}
	if base.WebCommitSignoffRequired != nil {
		out.WebCommitSignoffRequired = cur.WebCommitSignoffRequired
	}
	if base.AllowForking != nil {
		out.AllowForking = cur.AllowForking
	}
	if base.Topics != nil {
		out.Topics = cur.Topics
	}
}
