package bbolt

import (
	"time"

	"go.etcd.io/bbolt"

	"github.com/your-org/gitdex/internal/audit"
)

// AuditStore implements audit.AuditLedger using BBolt.
type AuditStore struct {
	db *bbolt.DB
}

// NewAuditStore creates a new AuditStore.
func NewAuditStore(db *bbolt.DB) *AuditStore {
	return &AuditStore{db: db}
}

func (s *AuditStore) Append(entry *audit.AuditEntry) error {
	if entry == nil {
		return ErrNilEntry
	}
	if entry.EntryID == "" {
		entry.EntryID = audit.GenerateEntryID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAuditEntries)
		if b == nil {
			return ErrBucketNotFound
		}
		data, err := jsonMarshal(entry)
		if err != nil {
			return err
		}
		return b.Put([]byte(entry.EntryID), data)
	})
}

func (s *AuditStore) Query(filters audit.AuditFilter) ([]*audit.AuditEntry, error) {
	var result []*audit.AuditEntry
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAuditEntries)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var e audit.AuditEntry
			if err := jsonUnmarshal(v, &e); err != nil {
				return err
			}
			if matchAuditFilter(&e, filters) {
				result = append(result, &e)
			}
			return nil
		})
	})
	return result, err
}

func (s *AuditStore) GetByCorrelation(correlationID string) ([]*audit.AuditEntry, error) {
	return s.Query(audit.AuditFilter{CorrelationID: correlationID})
}

func (s *AuditStore) GetByTask(taskID string) ([]*audit.AuditEntry, error) {
	return s.Query(audit.AuditFilter{TaskID: taskID})
}

func (s *AuditStore) GetByEntryID(entryID string) (*audit.AuditEntry, bool) {
	var entry *audit.AuditEntry
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAuditEntries)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(entryID))
		if v == nil {
			return nil
		}
		var e audit.AuditEntry
		if err := jsonUnmarshal(v, &e); err != nil {
			return err
		}
		entry = &e
		return nil
	})
	return entry, err == nil && entry != nil
}

func matchAuditFilter(e *audit.AuditEntry, f audit.AuditFilter) bool {
	if f.EntryID != "" && e.EntryID != f.EntryID {
		return false
	}
	if f.EventType != "" && e.EventType != f.EventType {
		return false
	}
	if f.TaskID != "" && e.TaskID != f.TaskID {
		return false
	}
	if f.CorrelationID != "" && e.CorrelationID != f.CorrelationID {
		return false
	}
	if f.FromTime != nil && e.Timestamp.Before(*f.FromTime) {
		return false
	}
	if f.ToTime != nil && e.Timestamp.After(*f.ToTime) {
		return false
	}
	return true
}
