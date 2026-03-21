package bbolt

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/autonomy"
)

// AutonomyStore implements autonomy.AutonomyStore using BBolt.
type AutonomyStore struct {
	db *bbolt.DB
}

// NewAutonomyStore creates a new AutonomyStore.
func NewAutonomyStore(db *bbolt.DB) *AutonomyStore {
	return &AutonomyStore{db: db}
}

func (s *AutonomyStore) SaveConfig(cfg *autonomy.AutonomyConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil config")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAutonomyConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		cp := *cfg
		if cp.ConfigID == "" {
			cp.ConfigID = "cfg_" + uuid.New().String()[:8]
		}
		if cp.CreatedAt.IsZero() {
			cp.CreatedAt = time.Now().UTC()
		}
		data, err := jsonMarshal(&cp)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(cp.ConfigID), data); err != nil {
			return err
		}

		activeB := tx.Bucket(bucketAutonomyConfigs)
		if activeB != nil {
			active := activeB.Get([]byte(ActiveKey))
			if active == nil || len(active) == 0 {
				_ = b.Put([]byte(ActiveKey), []byte(cp.ConfigID))
			}
		}

		cfg.ConfigID = cp.ConfigID
		cfg.CreatedAt = cp.CreatedAt
		return nil
	})
}

func (s *AutonomyStore) GetConfig(configID string) (*autonomy.AutonomyConfig, error) {
	var cfg *autonomy.AutonomyConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAutonomyConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(configID))
		if v == nil {
			return fmt.Errorf("config %q not found", configID)
		}
		var c autonomy.AutonomyConfig
		if err := jsonUnmarshal(v, &c); err != nil {
			return err
		}
		cfg = &c
		return nil
	})
	return cfg, err
}

func (s *AutonomyStore) GetActiveConfig() (*autonomy.AutonomyConfig, error) {
	var activeID []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAutonomyConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		activeID = b.Get([]byte(ActiveKey))
		return nil
	})
	if err != nil || activeID == nil || len(activeID) == 0 {
		return nil, nil
	}
	return s.GetConfig(string(activeID))
}

func (s *AutonomyStore) SetActiveConfig(configID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAutonomyConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		if b.Get([]byte(configID)) == nil {
			return fmt.Errorf("config %q not found", configID)
		}
		return b.Put([]byte(ActiveKey), []byte(configID))
	})
}

func (s *AutonomyStore) ListConfigs() ([]*autonomy.AutonomyConfig, error) {
	var result []*autonomy.AutonomyConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAutonomyConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(k, v []byte) error {
			if string(k) == ActiveKey {
				return nil
			}
			var c autonomy.AutonomyConfig
			if err := jsonUnmarshal(v, &c); err != nil {
				return err
			}
			result = append(result, &c)
			return nil
		})
	})
	return result, err
}
