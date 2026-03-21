package bbolt

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/collaboration"
)

// ObjectStore implements collaboration.ObjectStore using BBolt.
type ObjectStore struct {
	db *bbolt.DB
}

// NewObjectStore creates a new ObjectStore.
func NewObjectStore(db *bbolt.DB) *ObjectStore {
	return &ObjectStore{db: db}
}

func repoNumberKey(owner, repo string, number int) string {
	return fmt.Sprintf("%s/%s#%d", owner, repo, number)
}

func (s *ObjectStore) SaveObject(_ context.Context, obj *collaboration.CollaborationObject) error {
	if obj == nil {
		return fmt.Errorf("object cannot be nil")
	}
	if obj.ObjectID == "" {
		obj.ObjectID = uuid.New().String()
	}
	now := time.Now().UTC()
	obj.UpdatedAt = now
	if obj.CreatedAt.IsZero() {
		obj.CreatedAt = now
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		main := tx.Bucket(bucketCollaborationObjs)
		idx := tx.Bucket(bucketObjectsByRepoNumber)
		if main == nil || idx == nil {
			return ErrBucketNotFound
		}

		key := repoNumberKey(obj.RepoOwner, obj.RepoName, obj.Number)
		existingID := idx.Get([]byte(key))
		if existingID != nil && string(existingID) != obj.ObjectID {
			return fmt.Errorf("object with same repo and number already exists")
		}

		data, err := jsonMarshal(obj)
		if err != nil {
			return err
		}
		if err := main.Put([]byte(obj.ObjectID), data); err != nil {
			return err
		}
		return idx.Put([]byte(key), []byte(obj.ObjectID))
	})
}

func (s *ObjectStore) GetObject(_ context.Context, objectID string) (*collaboration.CollaborationObject, error) {
	var obj *collaboration.CollaborationObject
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketCollaborationObjs)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(objectID))
		if v == nil {
			return fmt.Errorf("object not found")
		}
		var o collaboration.CollaborationObject
		if err := jsonUnmarshal(v, &o); err != nil {
			return err
		}
		obj = &o
		return nil
	})
	return obj, err
}

func (s *ObjectStore) ListObjects(_ context.Context, filter *collaboration.ObjectFilter) ([]*collaboration.CollaborationObject, error) {
	var result []*collaboration.CollaborationObject
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketCollaborationObjs)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var o collaboration.CollaborationObject
			if err := jsonUnmarshal(v, &o); err != nil {
				return err
			}
			if filter != nil && !matchObjectFilter(&o, filter) {
				return nil
			}
			result = append(result, &o)
			return nil
		})
	})
	return result, err
}

func (s *ObjectStore) GetByRepoAndNumber(ctx context.Context, owner, repo string, number int) (*collaboration.CollaborationObject, error) {
	var objectID []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		idx := tx.Bucket(bucketObjectsByRepoNumber)
		if idx == nil {
			return ErrBucketNotFound
		}
		key := repoNumberKey(owner, repo, number)
		objectID = idx.Get([]byte(key))
		return nil
	})
	if err != nil || objectID == nil {
		return nil, fmt.Errorf("object not found")
	}
	return s.GetObject(ctx, string(objectID))
}

func matchObjectFilter(obj *collaboration.CollaborationObject, filter *collaboration.ObjectFilter) bool {
	if filter.ObjectType != "" && obj.ObjectType != filter.ObjectType {
		return false
	}
	if filter.State != "" && filter.State != "all" && filter.State != obj.State {
		return false
	}
	if filter.RepoOwner != "" && obj.RepoOwner != filter.RepoOwner {
		return false
	}
	if filter.RepoName != "" && obj.RepoName != filter.RepoName {
		return false
	}
	if len(filter.Labels) > 0 {
		hasAll := true
		for _, l := range filter.Labels {
			found := false
			for _, ol := range obj.Labels {
				if ol == l {
					found = true
					break
				}
			}
			if !found {
				hasAll = false
				break
			}
		}
		if !hasAll {
			return false
		}
	}
	if filter.Assignee != "" {
		found := false
		for _, a := range obj.Assignees {
			if a == filter.Assignee {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if filter.Author != "" && obj.Author != filter.Author {
		return false
	}
	if filter.Milestone != "" && obj.Milestone != filter.Milestone {
		return false
	}
	return true
}
