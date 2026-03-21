package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/campaign"
)

type CampaignStore struct {
	db *sql.DB
}

func NewCampaignStore(db *sql.DB) *CampaignStore {
	return &CampaignStore{db: db}
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

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO campaigns (campaign_id, name, description, status, target_repos, plan_template, policy_bundle_id, created_by, created_at, updated_at)
		VALUES (?, ?, NULLIF(?,''), ?, ?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), ?, ?)
		ON CONFLICT (campaign_id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			status = excluded.status,
			target_repos = excluded.target_repos,
			plan_template = excluded.plan_template,
			policy_bundle_id = excluded.policy_bundle_id,
			created_by = excluded.created_by,
			updated_at = excluded.updated_at
	`, c.CampaignID, c.Name, c.Description, c.Status, targetReposJSON, c.PlanTemplate, c.PolicyBundleID, c.CreatedBy, formatTime(c.CreatedAt), formatTime(c.UpdatedAt))
	return err
}

func (s *CampaignStore) GetCampaign(campaignID string) (*campaign.Campaign, error) {
	var c campaign.Campaign
	var targetReposJSON []byte
	var description, planTemplate, policyBundleID, createdBy sql.NullString
	var createdAtStr, updatedAtStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT campaign_id, name, description, status, target_repos, plan_template, policy_bundle_id, created_by, created_at, updated_at
		FROM campaigns WHERE campaign_id = ?
	`, campaignID).Scan(&c.CampaignID, &c.Name, &description, &c.Status, &targetReposJSON, &planTemplate, &policyBundleID, &createdBy, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("campaign %q not found", campaignID)
		}
		return nil, err
	}
	if description.Valid {
		c.Description = description.String
	}
	if planTemplate.Valid {
		c.PlanTemplate = planTemplate.String
	}
	if policyBundleID.Valid {
		c.PolicyBundleID = policyBundleID.String
	}
	if createdBy.Valid {
		c.CreatedBy = createdBy.String
	}
	_ = json.Unmarshal(targetReposJSON, &c.TargetRepos)
	c.CreatedAt, _ = parseTime(createdAtStr)
	c.UpdatedAt, _ = parseTime(updatedAtStr)
	return &c, nil
}

func (s *CampaignStore) ListCampaigns() ([]*campaign.Campaign, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT campaign_id FROM campaigns ORDER BY created_at`)
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

	res, err := s.db.ExecContext(ctx(), `
		UPDATE campaigns SET name = ?, description = NULLIF(?,''), status = ?, target_repos = ?, plan_template = NULLIF(?,''), policy_bundle_id = NULLIF(?,''), created_by = NULLIF(?,''), updated_at = ?
		WHERE campaign_id = ?
	`, c.Name, c.Description, c.Status, targetReposJSON, c.PlanTemplate, c.PolicyBundleID, c.CreatedBy, formatTime(c.UpdatedAt), c.CampaignID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("campaign %q not found", c.CampaignID)
	}
	return nil
}
