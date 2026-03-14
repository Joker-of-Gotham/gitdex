package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type prReviewExecutor struct{ client *Client }

func (e prReviewExecutor) CapabilityID() string { return "pr_review" }

func (e prReviewExecutor) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	pullNumber := normalizeScopeValue(req.Scope, "pull_number", "")
	view := strings.ToLower(normalizeScopeValue(req.Query, "view", ""))
	path := e.client.repoPath("/pulls")
	switch view {
	case "reviews":
		if pullNumber == "" {
			pullNumber = strings.TrimSpace(req.ResourceID)
		}
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		path = e.client.repoPath("/pulls/" + trimResourceID(pullNumber) + "/reviews")
	case "review":
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		if strings.TrimSpace(req.ResourceID) == "" {
			return nil, fmt.Errorf("review id is required")
		}
		path = e.client.repoPath("/pulls/" + trimResourceID(pullNumber) + "/reviews/" + trimResourceID(req.ResourceID))
	case "requested_reviewers":
		if pullNumber == "" {
			pullNumber = strings.TrimSpace(req.ResourceID)
		}
		if pullNumber == "" {
			return nil, fmt.Errorf("pull_number is required")
		}
		path = e.client.repoPath("/pulls/" + trimResourceID(pullNumber) + "/requested_reviewers")
	default:
		if pullNumber == "" {
			pullNumber = strings.TrimSpace(req.ResourceID)
		}
		if pullNumber != "" {
			path = e.client.repoPath("/pulls/" + trimResourceID(pullNumber))
		} else {
			path = appendQuery(path, req.Query, "view")
		}
	}
	raw, err := e.client.doRaw(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		resourceID = pullNumber
	}
	return snapshot(e.CapabilityID(), resourceID, raw), nil
}

func (e prReviewExecutor) Mutate(ctx context.Context, req platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	pullNumber := normalizeScopeValue(req.Scope, "pull_number", "")
	if pullNumber == "" {
		return nil, fmt.Errorf("pull_number is required")
	}
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	result := &platform.AdminMutationResult{
		CapabilityID: e.CapabilityID(),
		Operation:    op,
		ResourceID:   strings.TrimSpace(req.ResourceID),
		Metadata:     map[string]string{"pull_number": pullNumber},
	}
	if result.ResourceID != "" {
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{
			ResourceID: result.ResourceID,
			Scope:      map[string]string{"pull_number": pullNumber},
			Query:      map[string]string{"view": "review"},
		})
	}

	switch op {
	case "approve", "request_changes", "comment":
		body, err := rawObject(req.Payload)
		if err != nil {
			return nil, err
		}
		body["event"] = reviewEventForOperation(op)
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)+"/reviews"), body, http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.ResourceID = extractResourceID(raw, result.ResourceID)
		result.After = snapshot(e.CapabilityID(), result.ResourceID, raw)
	case "dismiss":
		reviewID := strings.TrimSpace(result.ResourceID)
		if reviewID == "" {
			reviewID = normalizeScopeValue(req.Scope, "review_id", "")
		}
		if reviewID == "" {
			return nil, fmt.Errorf("review id is required")
		}
		result.ResourceID = reviewID
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{
			ResourceID: reviewID,
			Scope:      map[string]string{"pull_number": pullNumber},
			Query:      map[string]string{"view": "review"},
		})
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)+"/reviews/"+trimResourceID(reviewID)+"/dismissals"), json.RawMessage(req.Payload), http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), reviewID, raw)
	case "request_reviewers", "remove_reviewers":
		path := e.client.repoPath("/pulls/" + trimResourceID(pullNumber) + "/requested_reviewers")
		result.Before, _ = e.Inspect(ctx, platform.AdminInspectRequest{
			Scope: map[string]string{"pull_number": pullNumber},
			Query: map[string]string{"view": "requested_reviewers"},
		})
		method := http.MethodPost
		if op == "remove_reviewers" {
			method = http.MethodDelete
		}
		raw, err := e.client.doRaw(ctx, method, path, json.RawMessage(req.Payload), http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		result.After = snapshot(e.CapabilityID(), pullNumber, raw)
		result.ResourceID = pullNumber
	default:
		return nil, fmt.Errorf("unsupported pr review operation: %s", op)
	}
	return result, nil
}

func (e prReviewExecutor) Validate(ctx context.Context, req platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	pullNumber := normalizeScopeValue(req.Scope, "pull_number", normalizeScopeValue(req.Mutation.Metadata, "pull_number", ""))
	switch req.Mutation.Operation {
	case "request_reviewers", "remove_reviewers":
		snap, err := e.Inspect(ctx, platform.AdminInspectRequest{
			ResourceID: pullNumber,
			Scope:      map[string]string{"pull_number": pullNumber},
			Query:      map[string]string{"view": "requested_reviewers"},
		})
		if err != nil {
			return &platform.AdminValidationResult{OK: false, Summary: err.Error(), ResourceID: pullNumber}, nil
		}
		ok, summary, err := validateRequestedReviewers(snap.State, req.Payload, req.Mutation.Operation)
		if err != nil {
			return nil, err
		}
		return &platform.AdminValidationResult{OK: ok, Summary: summary, ResourceID: pullNumber, Snapshot: snap}, nil
	default:
		return validateByInspect(ctx, reviewValidator{executor: e, pullNumber: pullNumber}, req, "pull request review validated")
	}
}

func (e prReviewExecutor) Rollback(ctx context.Context, req platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	if req.Mutation == nil {
		return nil, fmt.Errorf("mutation result is required")
	}
	pullNumber := normalizeScopeValue(req.Scope, "pull_number", normalizeScopeValue(req.Mutation.Metadata, "pull_number", ""))
	switch req.Mutation.Operation {
	case "approve", "request_changes", "comment":
		if strings.TrimSpace(req.Mutation.ResourceID) == "" {
			return nil, fmt.Errorf("review id is required for rollback")
		}
		payload := req.Payload
		if len(payload) == 0 {
			payload = json.RawMessage(`{"message":"dismissed by gitdex rollback"}`)
		}
		raw, err := e.client.doRaw(ctx, http.MethodPut, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)+"/reviews/"+trimResourceID(req.Mutation.ResourceID)+"/dismissals"), payload, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "review dismissed as rollback", Snapshot: snapshot(e.CapabilityID(), req.Mutation.ResourceID, raw)}, nil
	case "dismiss":
		if req.Mutation.Before == nil {
			return &platform.AdminRollbackResult{OK: false, Summary: "dismiss rollback requires the previous review snapshot"}, nil
		}
		restore, err := restoreReviewPayload(req.Mutation.Before.State)
		if err != nil {
			return nil, err
		}
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)+"/reviews"), restore, http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "equivalent review re-created", Snapshot: snapshot(e.CapabilityID(), extractResourceID(raw, ""), raw)}, nil
	case "request_reviewers":
		if _, err := e.Mutate(ctx, platform.AdminMutationRequest{
			Operation: "remove_reviewers",
			Scope:     map[string]string{"pull_number": pullNumber},
			Payload:   cloneRaw(req.Payload),
		}); err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "requested reviewers removed"}, nil
	case "remove_reviewers":
		if req.Mutation.Before == nil {
			return &platform.AdminRollbackResult{OK: false, Summary: "remove reviewers rollback requires previous requested reviewers"}, nil
		}
		payload, err := requestedReviewersPayload(req.Mutation.Before.State)
		if err != nil {
			return nil, err
		}
		raw, err := e.client.doRaw(ctx, http.MethodPost, e.client.repoPath("/pulls/"+trimResourceID(pullNumber)+"/requested_reviewers"), payload, http.StatusCreated, http.StatusOK)
		if err != nil {
			return nil, err
		}
		return &platform.AdminRollbackResult{OK: true, Summary: "requested reviewers restored", Snapshot: snapshot(e.CapabilityID(), pullNumber, raw)}, nil
	default:
		return &platform.AdminRollbackResult{OK: false, Summary: "unsupported rollback operation for pr review"}, nil
	}
}

type reviewValidator struct {
	executor   prReviewExecutor
	pullNumber string
}

func (r reviewValidator) CapabilityID() string { return r.executor.CapabilityID() }
func (r reviewValidator) Inspect(ctx context.Context, req platform.AdminInspectRequest) (*platform.AdminSnapshot, error) {
	return r.executor.Inspect(ctx, platform.AdminInspectRequest{
		ResourceID: req.ResourceID,
		Scope:      map[string]string{"pull_number": r.pullNumber},
		Query:      map[string]string{"view": "review"},
	})
}
func (r reviewValidator) Mutate(context.Context, platform.AdminMutationRequest) (*platform.AdminMutationResult, error) {
	return nil, fmt.Errorf("unsupported")
}
func (r reviewValidator) Validate(context.Context, platform.AdminValidationRequest) (*platform.AdminValidationResult, error) {
	return nil, fmt.Errorf("unsupported")
}
func (r reviewValidator) Rollback(context.Context, platform.AdminRollbackRequest) (*platform.AdminRollbackResult, error) {
	return nil, fmt.Errorf("unsupported")
}

func reviewEventForOperation(op string) string {
	switch op {
	case "approve":
		return "APPROVE"
	case "request_changes":
		return "REQUEST_CHANGES"
	default:
		return "COMMENT"
	}
}

func validateRequestedReviewers(actual json.RawMessage, expected json.RawMessage, operation string) (bool, string, error) {
	var current struct {
		Users []struct {
			Login string `json:"login"`
		} `json:"users"`
		Teams []struct {
			Slug string `json:"slug"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(actual, &current); err != nil {
		return false, "", err
	}
	var desired struct {
		Reviewers     []string `json:"reviewers"`
		TeamReviewers []string `json:"team_reviewers"`
	}
	if len(expected) > 0 {
		if err := json.Unmarshal(expected, &desired); err != nil {
			return false, "", err
		}
	}
	userSet := map[string]struct{}{}
	for _, user := range current.Users {
		userSet[strings.ToLower(strings.TrimSpace(user.Login))] = struct{}{}
	}
	teamSet := map[string]struct{}{}
	for _, team := range current.Teams {
		teamSet[strings.ToLower(strings.TrimSpace(team.Slug))] = struct{}{}
	}
	mustExist := !strings.EqualFold(operation, "remove_reviewers")
	for _, reviewer := range desired.Reviewers {
		_, ok := userSet[strings.ToLower(strings.TrimSpace(reviewer))]
		if ok != mustExist {
			if mustExist {
				return false, "requested reviewer missing", nil
			}
			return false, "requested reviewer still present", nil
		}
	}
	for _, team := range desired.TeamReviewers {
		_, ok := teamSet[strings.ToLower(strings.TrimSpace(team))]
		if ok != mustExist {
			if mustExist {
				return false, "requested team missing", nil
			}
			return false, "requested team still present", nil
		}
	}
	if mustExist {
		return true, "requested reviewers updated", nil
	}
	return true, "requested reviewers removed", nil
}

func restoreReviewPayload(raw json.RawMessage) (map[string]any, error) {
	obj, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	restore := map[string]any{}
	if body := strings.TrimSpace(stringValue(obj["body"])); body != "" {
		restore["body"] = body
	}
	switch strings.ToUpper(strings.TrimSpace(stringValue(obj["state"]))) {
	case "APPROVED":
		restore["event"] = "APPROVE"
	case "CHANGES_REQUESTED":
		restore["event"] = "REQUEST_CHANGES"
	default:
		restore["event"] = "COMMENT"
	}
	if commitID := strings.TrimSpace(stringValue(obj["commit_id"])); commitID != "" {
		restore["commit_id"] = commitID
	}
	return restore, nil
}

func requestedReviewersPayload(raw json.RawMessage) (json.RawMessage, error) {
	var current struct {
		Users []struct {
			Login string `json:"login"`
		} `json:"users"`
		Teams []struct {
			Slug string `json:"slug"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(raw, &current); err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if len(current.Users) > 0 {
		reviewers := make([]string, 0, len(current.Users))
		for _, user := range current.Users {
			if strings.TrimSpace(user.Login) != "" {
				reviewers = append(reviewers, strings.TrimSpace(user.Login))
			}
		}
		payload["reviewers"] = reviewers
	}
	if len(current.Teams) > 0 {
		teams := make([]string, 0, len(current.Teams))
		for _, team := range current.Teams {
			if strings.TrimSpace(team.Slug) != "" {
				teams = append(teams, strings.TrimSpace(team.Slug))
			}
		}
		payload["team_reviewers"] = teams
	}
	return marshalRaw(payload)
}
