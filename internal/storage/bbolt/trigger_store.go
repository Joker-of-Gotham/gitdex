package bbolt

import (
	"fmt"
	"sort"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/autonomy"
)

// TriggerStore implements autonomy.TriggerStore using BBolt.
type TriggerStore struct {
	db *bbolt.DB
}

// NewTriggerStore creates a new TriggerStore.
func NewTriggerStore(db *bbolt.DB) *TriggerStore {
	return &TriggerStore{db: db}
}

func (s *TriggerStore) SaveTrigger(cfg *autonomy.TriggerConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil trigger config")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTriggerConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		cp := *cfg
		if cp.TriggerID == "" {
			cp.TriggerID = "tr_" + uuid.New().String()[:8]
		}
		if cp.CreatedAt.IsZero() {
			cp.CreatedAt = time.Now().UTC()
		}
		data, err := jsonMarshal(&cp)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(cp.TriggerID), data); err != nil {
			return err
		}
		cfg.TriggerID = cp.TriggerID
		cfg.CreatedAt = cp.CreatedAt
		return nil
	})
}

func (s *TriggerStore) GetTrigger(triggerID string) (*autonomy.TriggerConfig, error) {
	var cfg *autonomy.TriggerConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTriggerConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(triggerID))
		if v == nil {
			return fmt.Errorf("trigger %q not found", triggerID)
		}
		var c autonomy.TriggerConfig
		if err := jsonUnmarshal(v, &c); err != nil {
			return err
		}
		cfg = &c
		return nil
	})
	return cfg, err
}

func (s *TriggerStore) ListTriggers() ([]*autonomy.TriggerConfig, error) {
	var result []*autonomy.TriggerConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTriggerConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var c autonomy.TriggerConfig
			if err := jsonUnmarshal(v, &c); err != nil {
				return err
			}
			result = append(result, &c)
			return nil
		})
	})
	return result, err
}

func (s *TriggerStore) EnableTrigger(triggerID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTriggerConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(triggerID))
		if v == nil {
			return fmt.Errorf("trigger %q not found", triggerID)
		}
		var c autonomy.TriggerConfig
		if err := jsonUnmarshal(v, &c); err != nil {
			return err
		}
		c.Enabled = true
		data, err := jsonMarshal(&c)
		if err != nil {
			return err
		}
		return b.Put([]byte(triggerID), data)
	})
}

func (s *TriggerStore) DisableTrigger(triggerID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTriggerConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(triggerID))
		if v == nil {
			return fmt.Errorf("trigger %q not found", triggerID)
		}
		var c autonomy.TriggerConfig
		if err := jsonUnmarshal(v, &c); err != nil {
			return err
		}
		c.Enabled = false
		data, err := jsonMarshal(&c)
		if err != nil {
			return err
		}
		return b.Put([]byte(triggerID), data)
	})
}

func (s *TriggerStore) AppendTriggerEvent(ev *autonomy.TriggerEvent) error {
	if ev == nil {
		return fmt.Errorf("cannot append nil trigger event")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTriggerEvents)
		if b == nil {
			return ErrBucketNotFound
		}
		cp := *ev
		if cp.EventID == "" {
			cp.EventID = "tev_" + uuid.New().String()[:8]
		}
		if cp.Timestamp.IsZero() {
			cp.Timestamp = time.Now().UTC()
		}
		data, err := jsonMarshal(&cp)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(cp.EventID), data); err != nil {
			return err
		}
		ev.EventID = cp.EventID
		ev.Timestamp = cp.Timestamp
		return nil
	})
}

func (s *TriggerStore) ListTriggerEvents(triggerID string, limit int) ([]*autonomy.TriggerEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	var result []*autonomy.TriggerEvent
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTriggerEvents)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var e autonomy.TriggerEvent
			if err := jsonUnmarshal(v, &e); err != nil {
				return err
			}
			if triggerID != "" && e.TriggerID != triggerID {
				return nil
			}
			result = append(result, &e)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
