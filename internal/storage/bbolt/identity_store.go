package bbolt

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/identity"
)

// IdentityStore implements identity.IdentityStore using BBolt.
type IdentityStore struct {
	db *bbolt.DB
}

// NewIdentityStore creates a new IdentityStore.
func NewIdentityStore(db *bbolt.DB) *IdentityStore {
	return &IdentityStore{db: db}
}

func (s *IdentityStore) SaveIdentity(id *identity.AppIdentity) error {
	if id == nil {
		return fmt.Errorf("cannot save nil identity")
	}
	if id.IdentityID == "" {
		id.IdentityID = "id_" + uuid.New().String()[:8]
	}
	if id.CreatedAt.IsZero() {
		id.CreatedAt = time.Now().UTC()
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketIdentities)
		if b == nil {
			return ErrBucketNotFound
		}
		data, err := jsonMarshal(id)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(id.IdentityID), data); err != nil {
			return err
		}
		// Set as current if no current identity yet
		current := b.Get([]byte(ActiveKey))
		if current == nil {
			return b.Put([]byte(ActiveKey), []byte(id.IdentityID))
		}
		return nil
	})
}

func (s *IdentityStore) GetIdentity(identityID string) (*identity.AppIdentity, error) {
	var id *identity.AppIdentity
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketIdentities)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(identityID))
		if v == nil {
			return fmt.Errorf("identity %q not found", identityID)
		}
		var a identity.AppIdentity
		if err := jsonUnmarshal(v, &a); err != nil {
			return err
		}
		id = &a
		return nil
	})
	return id, err
}

func (s *IdentityStore) ListIdentities() ([]*identity.AppIdentity, error) {
	var result []*identity.AppIdentity
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketIdentities)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(k, v []byte) error {
			if string(k) == ActiveKey {
				return nil
			}
			var a identity.AppIdentity
			if err := jsonUnmarshal(v, &a); err != nil {
				return err
			}
			result = append(result, &a)
			return nil
		})
	})
	return result, err
}

func (s *IdentityStore) GetCurrentIdentity() (*identity.AppIdentity, error) {
	var currentID string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketIdentities)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(ActiveKey))
		if v == nil {
			return nil
		}
		currentID = string(v)
		return nil
	})
	if err != nil || currentID == "" {
		return nil, err
	}
	return s.GetIdentity(currentID)
}

func (s *IdentityStore) SetCurrentIdentity(identityID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketIdentities)
		if b == nil {
			return ErrBucketNotFound
		}
		if b.Get([]byte(identityID)) == nil {
			return fmt.Errorf("identity %q not found", identityID)
		}
		return b.Put([]byte(ActiveKey), []byte(identityID))
	})
}
