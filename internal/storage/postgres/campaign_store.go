package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/campaign"
)

type CampaignStore struct {
	pool *pgxpool.Pool
}

func NewCampaignStore(pool *pgxpool.Pool) *CampaignStore {
	return &CampaignStore{pool: pool}
}

func (s *CampaignStore) SaveCampaign(c *campaign.Campaign) error {
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

	targetReposJSON, _ := json.Marshal(c.TargetRepos)

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO campaigns (campaign_id, name, description, status, target_repos, plan_template, policy_bundle_id, created_by, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3,''), $4, $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''), $9, $10)
		ON CONFLICT (campaign_id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			target_repos = EXCLUDED.target_repos,
			plan_template = EXCLUDED.plan_template,
			policy_bundle_id = EXCLUDED.policy_bundle_id,
			created_by = EXCLUDED.created_by,
			updated_at = EXCLUDED.updated_at
	`, c.CampaignID, c.Name, c.Description, c.Status, targetReposJSON, c.PlanTemplate, c.PolicyBundleID, c.CreatedBy, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *CampaignStore) GetCampaign(campaignID string) (*campaign.Campaign, error) {
	var c campaign.Campaign
	var targetReposJSON []byte
	var description, planTemplate, policyBundleID, createdBy *string

	err := s.pool.QueryRow(ctx(), `
		SELECT campaign_id, name, description, status, target_repos, plan_template, policy_bundle_id, created_by, created_at, updated_at
		FROM campaigns WHERE campaign_id = $1
	`, campaignID).Scan(&c.CampaignID, &c.Name, &description, &c.Status, &targetReposJSON, &planTemplate, &policyBundleID, &createdBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("campaign %q not found", campaignID)
		}
		return nil, err
	}
	if description != nil {
		c.Description = *description
	}
	if planTemplate != nil {
		c.PlanTemplate = *planTemplate
	}
	if policyBundleID != nil {
		c.PolicyBundleID = *policyBundleID
	}
	if createdBy != nil {
		c.CreatedBy = *createdBy
	}
	_ = json.Unmarshal(targetReposJSON, &c.TargetRepos)
	return &c, nil
}

func (s *CampaignStore) ListCampaigns() ([]*campaign.Campaign, error) {
	rows, err := s.pool.Query(ctx(), `SELECT campaign_id FROM campaigns ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*campaign.Campaign, 0, len(ids))
	for _, id := range ids {
		c, err := s.GetCampaign(id)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *CampaignStore) UpdateCampaign(c *campaign.Campaign) error {
	if c == nil {
		return fmt.Errorf("cannot update nil campaign")
	}
	c.UpdatedAt = time.Now().UTC()

	targetReposJSON, _ := json.Marshal(c.TargetRepos)

	cmd, err := s.pool.Exec(ctx(), `
		UPDATE campaigns SET name = $1, description = NULLIF($2,''), status = $3, target_repos = $4, plan_template = NULLIF($5,''), policy_bundle_id = NULLIF($6,''), created_by = NULLIF($7,''), updated_at = $8
		WHERE campaign_id = $9
	`, c.Name, c.Description, c.Status, targetReposJSON, c.PlanTemplate, c.PolicyBundleID, c.CreatedBy, c.UpdatedAt, c.CampaignID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("campaign %q not found", c.CampaignID)
	}
	return nil
}
