package campaign

import (
	"context"
	"fmt"
)

type InterventionType string

const (
	InterventionApproveRepo  InterventionType = "approve_repo"
	InterventionExcludeRepo  InterventionType = "exclude_repo"
	InterventionRetryRepo    InterventionType = "retry_repo"
	InterventionOverridePlan InterventionType = "override_plan"
	InterventionPauseRepo    InterventionType = "pause_repo"
	InterventionResumeRepo   InterventionType = "resume_repo"
)

type InterventionRequest struct {
	InterventionType InterventionType  `json:"intervention_type" yaml:"intervention_type"`
	CampaignID       string            `json:"campaign_id" yaml:"campaign_id"`
	Owner            string            `json:"owner" yaml:"owner"`
	Repo             string            `json:"repo" yaml:"repo"`
	Reason           string            `json:"reason,omitempty" yaml:"reason,omitempty"`
	Actor            string            `json:"actor" yaml:"actor"`
	Overrides        map[string]string `json:"overrides,omitempty" yaml:"overrides,omitempty"`
}

type InterventionResult struct {
	Request        InterventionRequest `json:"request" yaml:"request"`
	Success        bool                `json:"success" yaml:"success"`
	PreviousStatus string              `json:"previous_status,omitempty" yaml:"previous_status,omitempty"`
	NewStatus      string              `json:"new_status,omitempty" yaml:"new_status,omitempty"`
	Message        string              `json:"message" yaml:"message"`
}

type InterventionEngine interface {
	Execute(ctx context.Context, req InterventionRequest) (*InterventionResult, error)
}

type DefaultInterventionEngine struct {
	store CampaignStore
}

func NewDefaultInterventionEngine(store CampaignStore) *DefaultInterventionEngine {
	return &DefaultInterventionEngine{store: store}
}

func (e *DefaultInterventionEngine) Execute(ctx context.Context, req InterventionRequest) (*InterventionResult, error) {
	if req.CampaignID == "" {
		return &InterventionResult{
			Request: req,
			Success: false,
			Message: "campaign_id cannot be empty",
		}, nil
	}
	c, err := e.store.GetCampaign(req.CampaignID)
	if err != nil {
		return &InterventionResult{
			Request: req,
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var idx int = -1
	for i, t := range c.TargetRepos {
		if t.Owner == req.Owner && t.Repo == req.Repo {
			idx = i
			break
		}
	}
	if idx < 0 {
		return &InterventionResult{
			Request: req,
			Success: false,
			Message: fmt.Sprintf("repo %s/%s not found in campaign", req.Owner, req.Repo),
		}, nil
	}

	t := &c.TargetRepos[idx]
	prevStatus := string(t.InclusionStatus)
	result := &InterventionResult{Request: req, PreviousStatus: prevStatus}

	switch req.InterventionType {
	case InterventionApproveRepo:
		t.InclusionStatus = InclusionIncluded
		result.NewStatus = string(InclusionIncluded)
		result.Success = true
		result.Message = fmt.Sprintf("approved %s/%s", req.Owner, req.Repo)
	case InterventionExcludeRepo:
		t.InclusionStatus = InclusionExcluded
		result.NewStatus = string(InclusionExcluded)
		result.Success = true
		result.Message = fmt.Sprintf("excluded %s/%s", req.Owner, req.Repo)
		if req.Reason != "" {
			result.Message = result.Message + ": " + req.Reason
		}
	case InterventionRetryRepo:
		t.InclusionStatus = InclusionPending
		result.NewStatus = string(InclusionPending)
		result.Success = true
		result.Message = fmt.Sprintf("retry scheduled for %s/%s", req.Owner, req.Repo)
	case InterventionOverridePlan:
		if len(req.Overrides) > 0 {
			if t.PerRepoOverrides == nil {
				t.PerRepoOverrides = make(map[string]string)
			}
			for k, v := range req.Overrides {
				t.PerRepoOverrides[k] = v
			}
		}
		result.NewStatus = prevStatus
		result.Success = true
		result.Message = fmt.Sprintf("overrides applied for %s/%s", req.Owner, req.Repo)
	case InterventionPauseRepo:
		if c.Status == StatusExecuting {
			c.Status = StatusPaused
		}
		result.NewStatus = prevStatus
		result.Success = true
		result.Message = fmt.Sprintf("paused campaign for %s/%s", req.Owner, req.Repo)
	case InterventionResumeRepo:
		if c.Status == StatusPaused {
			c.Status = StatusExecuting
		}
		result.NewStatus = prevStatus
		result.Success = true
		result.Message = fmt.Sprintf("resumed campaign for %s/%s", req.Owner, req.Repo)
	default:
		result.Success = false
		result.Message = fmt.Sprintf("unknown intervention type: %s", req.InterventionType)
		return result, nil
	}

	if result.Success {
		if err := e.store.UpdateCampaign(c); err != nil {
			result.Success = false
			result.Message = err.Error()
			return result, nil
		}
	}

	return result, nil
}
