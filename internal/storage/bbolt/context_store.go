package bbolt

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/collaboration"
)

// ContextStore implements collaboration.ContextStore using BBolt.
type ContextStore struct {
	db *bbolt.DB
}

// NewContextStore creates a new ContextStore.
func NewContextStore(db *bbolt.DB) *ContextStore {
	return &ContextStore{db: db}
}

func (s *ContextStore) SaveContext(_ context.Context, tc *collaboration.TaskContext) error {
	if tc == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if tc.ContextID == "" {
		tc.ContextID = uuid.New().String()
	}
	tc.CreatedAt = time.Now().UTC()

	return s.db.Update(func(tx *bbolt.Tx) error {
		main := tx.Bucket(bucketTaskContexts)
		idx := tx.Bucket(bucketContextsByObjectRef)
		if main == nil || idx == nil {
			return ErrBucketNotFound
		}

		// If overwriting by primary ref, remove old context from byID if different
		existingID := idx.Get([]byte(tc.PrimaryObjectRef))
		if existingID != nil && string(existingID) != tc.ContextID {
			_ = main.Delete(existingID)
		}

		data, err := jsonMarshal(tc)
		if err != nil {
			return err
		}
		if err := main.Put([]byte(tc.ContextID), data); err != nil {
			return err
		}
		return idx.Put([]byte(tc.PrimaryObjectRef), []byte(tc.ContextID))
	})
}

func (s *ContextStore) GetContext(_ context.Context, contextID string) (*collaboration.TaskContext, error) {
	var tc *collaboration.TaskContext
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTaskContexts)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(contextID))
		if v == nil {
			return fmt.Errorf("context not found")
		}
		var c collaboration.TaskContext
		if err := jsonUnmarshal(v, &c); err != nil {
			return err
		}
		tc = &c
		return nil
	})
	return tc, err
}

func (s *ContextStore) ListContexts(_ context.Context) ([]*collaboration.TaskContext, error) {
	var result []*collaboration.TaskContext
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTaskContexts)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var c collaboration.TaskContext
			if err := jsonUnmarshal(v, &c); err != nil {
				return err
			}
			result = append(result, &c)
			return nil
		})
	})
	return result, err
}

func (s *ContextStore) GetByObjectRef(_ context.Context, objectRef string) (*collaboration.TaskContext, error) {
	var contextID []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		idx := tx.Bucket(bucketContextsByObjectRef)
		if idx == nil {
			return ErrBucketNotFound
		}
		contextID = idx.Get([]byte(objectRef))
		return nil
	})
	if err != nil || contextID == nil {
		return nil, fmt.Errorf("context not found for object %q", objectRef)
	}
	return s.GetContext(nil, string(contextID))
}
