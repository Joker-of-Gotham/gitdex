package bbolt

import (
	"fmt"
	"sort"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/autonomy"
)

// MonitorStore implements autonomy.MonitorStore using BBolt.
type MonitorStore struct {
	db *bbolt.DB
}

// NewMonitorStore creates a new MonitorStore.
func NewMonitorStore(db *bbolt.DB) *MonitorStore {
	return &MonitorStore{db: db}
}

func (s *MonitorStore) SaveMonitorConfig(cfg *autonomy.MonitorConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil monitor config")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMonitorConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		cp := *cfg
		if cp.MonitorID == "" {
			cp.MonitorID = "mon_" + uuid.New().String()[:8]
		}
		data, err := jsonMarshal(&cp)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(cp.MonitorID), data); err != nil {
			return err
		}
		cfg.MonitorID = cp.MonitorID
		return nil
	})
}

func (s *MonitorStore) GetMonitorConfig(monitorID string) (*autonomy.MonitorConfig, error) {
	var cfg *autonomy.MonitorConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMonitorConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(monitorID))
		if v == nil {
			return fmt.Errorf("monitor %q not found", monitorID)
		}
		var c autonomy.MonitorConfig
		if err := jsonUnmarshal(v, &c); err != nil {
			return err
		}
		cfg = &c
		return nil
	})
	return cfg, err
}

func (s *MonitorStore) ListMonitorConfigs() ([]*autonomy.MonitorConfig, error) {
	var result []*autonomy.MonitorConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMonitorConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var c autonomy.MonitorConfig
			if err := jsonUnmarshal(v, &c); err != nil {
				return err
			}
			result = append(result, &c)
			return nil
		})
	})
	return result, err
}

func (s *MonitorStore) AppendEvent(ev *autonomy.MonitorEvent) error {
	if ev == nil {
		return fmt.Errorf("cannot append nil event")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMonitorEvents)
		if b == nil {
			return ErrBucketNotFound
		}
		cp := *ev
		if cp.EventID == "" {
			cp.EventID = "ev_" + uuid.New().String()[:8]
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

func (s *MonitorStore) ListEvents(filter autonomy.MonitorEventFilter) ([]*autonomy.MonitorEvent, error) {
	var result []*autonomy.MonitorEvent
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMonitorEvents)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var e autonomy.MonitorEvent
			if err := jsonUnmarshal(v, &e); err != nil {
				return err
			}
			if filter.MonitorID != "" && e.MonitorID != filter.MonitorID {
				return nil
			}
			if filter.RepoOwner != "" && e.RepoOwner != filter.RepoOwner {
				return nil
			}
			if filter.RepoName != "" && e.RepoName != filter.RepoName {
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

func (s *MonitorStore) RemoveMonitorConfig(monitorID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMonitorConfigs)
		if b == nil {
			return ErrBucketNotFound
		}
		if b.Get([]byte(monitorID)) == nil {
			return fmt.Errorf("monitor %q not found", monitorID)
		}
		return b.Delete([]byte(monitorID))
	})
}
