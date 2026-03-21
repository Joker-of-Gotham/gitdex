package campaign

import (
	"context"
	"fmt"
	"time"
)

type RepoStatus string

const (
	RepoStatusNotStarted       RepoStatus = "not_started"
	RepoStatusPlanning         RepoStatus = "planning"
	RepoStatusAwaitingApproval RepoStatus = "awaiting_approval"
	RepoStatusApproved         RepoStatus = "approved"
	RepoStatusExecuting        RepoStatus = "executing"
	RepoStatusSucceeded        RepoStatus = "succeeded"
	RepoStatusFailed           RepoStatus = "failed"
	RepoStatusExcluded         RepoStatus = "excluded"
)

type MatrixEntry struct {
	Owner       string     `json:"owner" yaml:"owner"`
	Repo        string     `json:"repo" yaml:"repo"`
	Status      RepoStatus `json:"status" yaml:"status"`
	PlanID      string     `json:"plan_id,omitempty" yaml:"plan_id,omitempty"`
	TaskID      string     `json:"task_id,omitempty" yaml:"task_id,omitempty"`
	LastUpdated time.Time  `json:"last_updated" yaml:"last_updated"`
	Notes       string     `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type MatrixSummary struct {
	Total     int `json:"total" yaml:"total"`
	Succeeded int `json:"succeeded" yaml:"succeeded"`
	Failed    int `json:"failed" yaml:"failed"`
	Pending   int `json:"pending" yaml:"pending"`
	Excluded  int `json:"excluded" yaml:"excluded"`
}

type CampaignMatrix struct {
	CampaignID string        `json:"campaign_id" yaml:"campaign_id"`
	Entries    []MatrixEntry `json:"entries" yaml:"entries"`
	Summary    MatrixSummary `json:"summary" yaml:"summary"`
}

type MatrixEngine interface {
	Build(ctx context.Context, campaign *Campaign) (*CampaignMatrix, error)
}

type DefaultMatrixEngine struct{}

func NewDefaultMatrixEngine() *DefaultMatrixEngine {
	return &DefaultMatrixEngine{}
}

func (e *DefaultMatrixEngine) Build(ctx context.Context, c *Campaign) (*CampaignMatrix, error) {
	if c == nil {
		return nil, fmt.Errorf("campaign cannot be nil")
	}
	entries := make([]MatrixEntry, 0, len(c.TargetRepos))
	summary := MatrixSummary{Total: len(c.TargetRepos)}

	now := time.Now().UTC()
	for _, t := range c.TargetRepos {
		var status RepoStatus
		switch t.InclusionStatus {
		case InclusionExcluded:
			status = RepoStatusExcluded
			summary.Excluded++
		case InclusionIncluded:
			status = RepoStatusSucceeded
			summary.Succeeded++
		default:
			status = RepoStatusAwaitingApproval
			summary.Pending++
		}
		entries = append(entries, MatrixEntry{
			Owner:       t.Owner,
			Repo:        t.Repo,
			Status:      status,
			PlanID:      "plan_" + c.CampaignID,
			TaskID:      "task_" + t.Repo,
			LastUpdated: now,
			Notes:       "",
		})
	}

	return &CampaignMatrix{
		CampaignID: c.CampaignID,
		Entries:    entries,
		Summary:    summary,
	}, nil
}
