package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v84/github"
)

// GetBranchProtection returns branch protection summary for a branch.
func (c *Client) GetBranchProtection(ctx context.Context, owner, repo, branch string) (*BranchProtection, error) {
	prot, resp, err := c.gh.Repositories.GetBranchProtection(ctx, owner, repo, branch)
	if err != nil {
		return nil, fmt.Errorf("github: get branch protection: %w", err)
	}
	logRateLimit(resp)
	return branchProtectionFromGH(prot), nil
}

// UpdateBranchProtection applies settings on top of the current protection configuration.
func (c *Client) UpdateBranchProtection(ctx context.Context, owner, repo, branch string, settings BranchProtectionSettings) error {
	prot, resp, err := c.gh.Repositories.GetBranchProtection(ctx, owner, repo, branch)
	if err != nil {
		return fmt.Errorf("github: get branch protection: %w", err)
	}
	logRateLimit(resp)

	req := protectionToRequest(prot)
	req.EnforceAdmins = settings.EnforceAdmins
	if req.RequiredPullRequestReviews == nil {
		req.RequiredPullRequestReviews = &gh.PullRequestReviewsEnforcementRequest{}
	}
	req.RequiredPullRequestReviews.RequiredApprovingReviewCount = settings.RequiredReviews
	req.RequireLinearHistory = gh.Bool(settings.RequireLinearHistory)

	_, resp, err = c.gh.Repositories.UpdateBranchProtection(ctx, owner, repo, branch, req)
	if err != nil {
		return fmt.Errorf("github: update branch protection: %w", err)
	}
	logRateLimit(resp)

	curSig := false
	if prot.RequiredSignatures != nil && prot.RequiredSignatures.Enabled != nil {
		curSig = *prot.RequiredSignatures.Enabled
	}
	if settings.RequireSignedCommits == curSig {
		return nil
	}
	if settings.RequireSignedCommits {
		_, resp, err := c.gh.Repositories.RequireSignaturesOnProtectedBranch(ctx, owner, repo, branch)
		if err != nil {
			return fmt.Errorf("github: require signed commits: %w", err)
		}
		logRateLimit(resp)
		return nil
	}
	resp, err = c.gh.Repositories.OptionalSignaturesOnProtectedBranch(ctx, owner, repo, branch)
	if err != nil {
		return fmt.Errorf("github: disable signed commits requirement: %w", err)
	}
	logRateLimit(resp)
	return nil
}

// ListRulesets lists repository rulesets (GET /repos/{owner}/{repo}/rulesets).
func (c *Client) ListRulesets(ctx context.Context, owner, repo string) ([]Ruleset, error) {
	rs, resp, err := c.gh.Repositories.GetAllRulesets(ctx, owner, repo, nil)
	if err != nil {
		return nil, fmt.Errorf("github: list rulesets: %w", err)
	}
	logRateLimit(resp)
	out := make([]Ruleset, 0, len(rs))
	for _, r := range rs {
		if r == nil {
			continue
		}
		target := ""
		if r.Target != nil {
			target = string(*r.Target)
		}
		id := int64(0)
		if r.ID != nil {
			id = *r.ID
		}
		out = append(out, Ruleset{
			ID:          id,
			Name:        r.Name,
			Target:      target,
			Enforcement: string(r.Enforcement),
		})
	}
	return out, nil
}

func branchProtectionFromGH(p *gh.Protection) *BranchProtection {
	if p == nil {
		return nil
	}
	bp := &BranchProtection{}
	if p.RequiredPullRequestReviews != nil {
		bp.RequiredReviews = p.RequiredPullRequestReviews.RequiredApprovingReviewCount
	}
	if p.RequiredSignatures != nil && p.RequiredSignatures.Enabled != nil {
		bp.RequireSignedCommits = *p.RequiredSignatures.Enabled
	}
	if p.EnforceAdmins != nil {
		bp.EnforceAdmins = p.EnforceAdmins.Enabled
	}
	if p.RequireLinearHistory != nil {
		bp.RequireLinearHistory = p.RequireLinearHistory.Enabled
	}
	return bp
}

func protectionToRequest(p *gh.Protection) *gh.ProtectionRequest {
	if p == nil {
		return &gh.ProtectionRequest{}
	}
	req := &gh.ProtectionRequest{
		RequiredStatusChecks:       p.RequiredStatusChecks,
		RequiredPullRequestReviews: pullReviewsToRequest(p.RequiredPullRequestReviews),
		Restrictions:               restrictionsToRequest(p.Restrictions),
	}
	if p.EnforceAdmins != nil {
		req.EnforceAdmins = p.EnforceAdmins.Enabled
	}
	if p.RequireLinearHistory != nil {
		v := p.RequireLinearHistory.Enabled
		req.RequireLinearHistory = &v
	}
	if p.AllowForcePushes != nil {
		v := p.AllowForcePushes.Enabled
		req.AllowForcePushes = &v
	}
	if p.AllowDeletions != nil {
		v := p.AllowDeletions.Enabled
		req.AllowDeletions = &v
	}
	if p.RequiredConversationResolution != nil {
		v := p.RequiredConversationResolution.Enabled
		req.RequiredConversationResolution = &v
	}
	if p.BlockCreations != nil && p.BlockCreations.Enabled != nil {
		req.BlockCreations = p.BlockCreations.Enabled
	}
	if p.LockBranch != nil && p.LockBranch.Enabled != nil {
		req.LockBranch = p.LockBranch.Enabled
	}
	if p.AllowForkSyncing != nil && p.AllowForkSyncing.Enabled != nil {
		req.AllowForkSyncing = p.AllowForkSyncing.Enabled
	}
	return req
}

func restrictionsToRequest(br *gh.BranchRestrictions) *gh.BranchRestrictionsRequest {
	if br == nil {
		return nil
	}
	return &gh.BranchRestrictionsRequest{
		Users: userLogins(br.Users),
		Teams: teamSlugs(br.Teams),
		Apps:  appSlugs(br.Apps),
	}
}

func pullReviewsToRequest(e *gh.PullRequestReviewsEnforcement) *gh.PullRequestReviewsEnforcementRequest {
	if e == nil {
		return nil
	}
	var bypass *gh.BypassPullRequestAllowancesRequest
	if e.BypassPullRequestAllowances != nil {
		b := e.BypassPullRequestAllowances
		bypass = &gh.BypassPullRequestAllowancesRequest{
			Users: userLogins(b.Users),
			Teams: teamSlugs(b.Teams),
			Apps:  appSlugs(b.Apps),
		}
	}
	var dismiss *gh.DismissalRestrictionsRequest
	if e.DismissalRestrictions != nil {
		d := e.DismissalRestrictions
		users := userLogins(d.Users)
		teams := teamSlugs(d.Teams)
		apps := appSlugs(d.Apps)
		dismiss = &gh.DismissalRestrictionsRequest{}
		if len(users) > 0 {
			dismiss.Users = &users
		}
		if len(teams) > 0 {
			dismiss.Teams = &teams
		}
		if len(apps) > 0 {
			dismiss.Apps = &apps
		}
		if dismiss.Users == nil && dismiss.Teams == nil && dismiss.Apps == nil {
			dismiss = nil
		}
	}
	rlp := e.RequireLastPushApproval
	return &gh.PullRequestReviewsEnforcementRequest{
		BypassPullRequestAllowancesRequest: bypass,
		DismissalRestrictionsRequest:       dismiss,
		DismissStaleReviews:                e.DismissStaleReviews,
		RequireCodeOwnerReviews:            e.RequireCodeOwnerReviews,
		RequiredApprovingReviewCount:       e.RequiredApprovingReviewCount,
		RequireLastPushApproval:            &rlp,
	}
}

func userLogins(users []*gh.User) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		if u != nil {
			out = append(out, u.GetLogin())
		}
	}
	return out
}

func teamSlugs(teams []*gh.Team) []string {
	out := make([]string, 0, len(teams))
	for _, t := range teams {
		if t != nil {
			out = append(out, t.GetSlug())
		}
	}
	return out
}

func appSlugs(apps []*gh.App) []string {
	out := make([]string, 0, len(apps))
	for _, a := range apps {
		if a != nil {
			out = append(out, a.GetSlug())
		}
	}
	return out
}
