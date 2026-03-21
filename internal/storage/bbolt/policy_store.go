package bbolt

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/policy"
)

// PolicyStore implements policy.PolicyBundleStore using BBolt.
type PolicyStore struct {
	db *bbolt.DB
}

// NewPolicyStore creates a new PolicyStore.
func NewPolicyStore(db *bbolt.DB) *PolicyStore {
	return &PolicyStore{db: db}
}

func (s *PolicyStore) SaveBundle(bundle *policy.PolicyBundle) error {
	if bundle == nil {
		return fmt.Errorf("cannot save nil bundle")
	}
	if bundle.BundleID == "" {
		bundle.BundleID = "bundle_" + uuid.New().String()[:8]
	}
	if bundle.Version == "" {
		bundle.Version = "1.0.0"
	}
	if bundle.CreatedAt.IsZero() {
		bundle.CreatedAt = time.Now().UTC()
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPolicyBundles)
		if b == nil {
			return ErrBucketNotFound
		}
		data, err := jsonMarshal(bundle)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(bundle.BundleID), data); err != nil {
			return err
		}
		// Set as active if no active bundle yet
		active := b.Get([]byte(ActiveKey))
		if active == nil {
			return b.Put([]byte(ActiveKey), []byte(bundle.BundleID))
		}
		return nil
	})
}

func (s *PolicyStore) GetBundle(bundleID string) (*policy.PolicyBundle, error) {
	var bundle *policy.PolicyBundle
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPolicyBundles)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(bundleID))
		if v == nil {
			return fmt.Errorf("bundle %q not found", bundleID)
		}
		var p policy.PolicyBundle
		if err := jsonUnmarshal(v, &p); err != nil {
			return err
		}
		bundle = &p
		return nil
	})
	return bundle, err
}

func (s *PolicyStore) ListBundles() ([]*policy.PolicyBundle, error) {
	var result []*policy.PolicyBundle
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPolicyBundles)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(k, v []byte) error {
			if string(k) == ActiveKey {
				return nil
			}
			var p policy.PolicyBundle
			if err := jsonUnmarshal(v, &p); err != nil {
				return err
			}
			result = append(result, &p)
			return nil
		})
	})
	return result, err
}

func (s *PolicyStore) GetActiveBundle() (*policy.PolicyBundle, error) {
	var activeID string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPolicyBundles)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(ActiveKey))
		if v == nil {
			return nil
		}
		activeID = string(v)
		return nil
	})
	if err != nil || activeID == "" {
		return nil, err
	}
	return s.GetBundle(activeID)
}

func (s *PolicyStore) SetActiveBundle(bundleID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPolicyBundles)
		if b == nil {
			return ErrBucketNotFound
		}
		if b.Get([]byte(bundleID)) == nil {
			return fmt.Errorf("bundle %q not found", bundleID)
		}
		return b.Put([]byte(ActiveKey), []byte(bundleID))
	})
}
