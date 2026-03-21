package bbolt

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/campaign"
)

// CampaignStore implements campaign.CampaignStore using BBolt.
type CampaignStore struct {
	db *bbolt.DB
}

// NewCampaignStore creates a new CampaignStore.
func NewCampaignStore(db *bbolt.DB) *CampaignStore {
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

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketCampaigns)
		if b == nil {
			return ErrBucketNotFound
		}
		data, err := jsonMarshal(c)
		if err != nil {
			return err
		}
		return b.Put([]byte(c.CampaignID), data)
	})
}

func (s *CampaignStore) GetCampaign(campaignID string) (*campaign.Campaign, error) {
	var c *campaign.Campaign
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketCampaigns)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(campaignID))
		if v == nil {
			return fmt.Errorf("campaign %q not found", campaignID)
		}
		var camp campaign.Campaign
		if err := jsonUnmarshal(v, &camp); err != nil {
			return err
		}
		c = &camp
		return nil
	})
	return c, err
}

func (s *CampaignStore) ListCampaigns() ([]*campaign.Campaign, error) {
	var result []*campaign.Campaign
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketCampaigns)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var c campaign.Campaign
			if err := jsonUnmarshal(v, &c); err != nil {
				return err
			}
			result = append(result, &c)
			return nil
		})
	})
	return result, err
}

func (s *CampaignStore) UpdateCampaign(c *campaign.Campaign) error {
	if c == nil {
		return fmt.Errorf("cannot update nil campaign")
	}
	c.UpdatedAt = time.Now().UTC()

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketCampaigns)
		if b == nil {
			return ErrBucketNotFound
		}
		if b.Get([]byte(c.CampaignID)) == nil {
			return fmt.Errorf("campaign %q not found", c.CampaignID)
		}
		data, err := jsonMarshal(c)
		if err != nil {
			return err
		}
		return b.Put([]byte(c.CampaignID), data)
	})
}
