package campaign

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CampaignStatus string

const (
	StatusDraft     CampaignStatus = "draft"
	StatusPlanning  CampaignStatus = "planning"
	StatusExecuting CampaignStatus = "executing"
	StatusPaused    CampaignStatus = "paused"
	StatusCompleted CampaignStatus = "completed"
	StatusCancelled CampaignStatus = "cancelled"
)

type InclusionStatus string

const (
	InclusionIncluded InclusionStatus = "included"
	InclusionExcluded InclusionStatus = "excluded"
	InclusionPending  InclusionStatus = "pending"
)

type RepoTarget struct {
	Owner            string            `json:"owner" yaml:"owner"`
	Repo             string            `json:"repo" yaml:"repo"`
	InclusionStatus  InclusionStatus   `json:"inclusion_status" yaml:"inclusion_status"`
	PerRepoOverrides map[string]string `json:"per_repo_overrides,omitempty" yaml:"per_repo_overrides,omitempty"`
}

type Campaign struct {
	CampaignID     string         `json:"campaign_id" yaml:"campaign_id"`
	Name           string         `json:"name" yaml:"name"`
	Description    string         `json:"description" yaml:"description"`
	Status         CampaignStatus `json:"status" yaml:"status"`
	TargetRepos    []RepoTarget   `json:"target_repos" yaml:"target_repos"`
	PlanTemplate   string         `json:"plan_template,omitempty" yaml:"plan_template,omitempty"`
	PolicyBundleID string         `json:"policy_bundle_id,omitempty" yaml:"policy_bundle_id,omitempty"`
	CreatedBy      string         `json:"created_by" yaml:"created_by"`
	CreatedAt      time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" yaml:"updated_at"`
}

type CampaignStore interface {
	SaveCampaign(c *Campaign) error
	GetCampaign(campaignID string) (*Campaign, error)
	ListCampaigns() ([]*Campaign, error)
	UpdateCampaign(c *Campaign) error
}

type MemoryCampaignStore struct {
	mu   sync.RWMutex
	byID map[string]*Campaign
}

func NewMemoryCampaignStore() *MemoryCampaignStore {
	return &MemoryCampaignStore{
		byID: make(map[string]*Campaign),
	}
}

func (s *MemoryCampaignStore) SaveCampaign(c *Campaign) error {
	if c == nil {
		return fmt.Errorf("cannot save nil campaign")
	}
	if len(c.TargetRepos) > 0 {
		seen := make(map[string]struct{})
		for _, t := range c.TargetRepos {
			key := t.Owner + "/" + t.Repo
			if _, ok := seen[key]; ok {
				return fmt.Errorf("duplicate repo in target_repos: %s", key)
			}
			seen[key] = struct{}{}
		}
	}
	if c.CampaignID == "" {
		c.CampaignID = "camp_" + uuid.New().String()[:8]
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := copyCampaign(c)
	s.byID[cp.CampaignID] = cp
	return nil
}

func (s *MemoryCampaignStore) GetCampaign(campaignID string) (*Campaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.byID[campaignID]
	if !ok {
		return nil, fmt.Errorf("campaign %q not found", campaignID)
	}
	return copyCampaign(c), nil
}

func (s *MemoryCampaignStore) ListCampaigns() ([]*Campaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Campaign, 0, len(s.byID))
	for _, c := range s.byID {
		result = append(result, copyCampaign(c))
	}
	return result, nil
}

func (s *MemoryCampaignStore) UpdateCampaign(c *Campaign) error {
	if c == nil {
		return fmt.Errorf("cannot update nil campaign")
	}
	c.UpdatedAt = time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[c.CampaignID]; !ok {
		return fmt.Errorf("campaign %q not found", c.CampaignID)
	}
	s.byID[c.CampaignID] = copyCampaign(c)
	return nil
}

func copyCampaign(c *Campaign) *Campaign {
	cp := *c
	if len(c.TargetRepos) > 0 {
		cp.TargetRepos = make([]RepoTarget, len(c.TargetRepos))
		copy(cp.TargetRepos, c.TargetRepos)
		for i := range c.TargetRepos {
			if c.TargetRepos[i].PerRepoOverrides != nil {
				cp.TargetRepos[i].PerRepoOverrides = make(map[string]string, len(c.TargetRepos[i].PerRepoOverrides))
				for k, v := range c.TargetRepos[i].PerRepoOverrides {
					cp.TargetRepos[i].PerRepoOverrides[k] = v
				}
			}
		}
	}
	return &cp
}
