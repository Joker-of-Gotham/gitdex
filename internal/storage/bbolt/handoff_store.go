package bbolt

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/autonomy"
)

// HandoffStore implements autonomy.HandoffStore using BBolt.
type HandoffStore struct {
	db *bbolt.DB
}

// NewHandoffStore creates a new HandoffStore.
func NewHandoffStore(db *bbolt.DB) *HandoffStore {
	return &HandoffStore{db: db}
}

func (s *HandoffStore) SavePackage(pkg *autonomy.HandoffPackage) error {
	if pkg == nil {
		return fmt.Errorf("cannot save nil package")
	}
	if pkg.TaskID == "" {
		return fmt.Errorf("TaskID cannot be empty")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		main := tx.Bucket(bucketHandoffPackages)
		idx := tx.Bucket(bucketHandoffsByTaskID)
		if main == nil || idx == nil {
			return ErrBucketNotFound
		}
		cp := copyHandoffPackage(pkg)
		if cp.PackageID == "" {
			cp.PackageID = "pkg_" + uuid.New().String()[:8]
		}
		if cp.CreatedAt.IsZero() {
			cp.CreatedAt = time.Now().UTC()
		}
		data, err := jsonMarshal(cp)
		if err != nil {
			return err
		}
		if err := main.Put([]byte(cp.PackageID), data); err != nil {
			return err
		}
		if err := idx.Put([]byte(cp.TaskID), []byte(cp.PackageID)); err != nil {
			return err
		}
		pkg.PackageID = cp.PackageID
		pkg.CreatedAt = cp.CreatedAt
		return nil
	})
}

func copyHandoffPackage(pkg *autonomy.HandoffPackage) *autonomy.HandoffPackage {
	cp := *pkg
	if len(pkg.CompletedSteps) > 0 {
		cp.CompletedSteps = make([]string, len(pkg.CompletedSteps))
		copy(cp.CompletedSteps, pkg.CompletedSteps)
	}
	if len(pkg.PendingSteps) > 0 {
		cp.PendingSteps = make([]string, len(pkg.PendingSteps))
		copy(cp.PendingSteps, pkg.PendingSteps)
	}
	if len(pkg.ContextData) > 0 {
		cp.ContextData = make(map[string]string, len(pkg.ContextData))
		for k, v := range pkg.ContextData {
			cp.ContextData[k] = v
		}
	}
	if len(pkg.Artifacts) > 0 {
		cp.Artifacts = make([]string, len(pkg.Artifacts))
		copy(cp.Artifacts, pkg.Artifacts)
	}
	if len(pkg.Recommendations) > 0 {
		cp.Recommendations = make([]string, len(pkg.Recommendations))
		copy(cp.Recommendations, pkg.Recommendations)
	}
	return &cp
}

func (s *HandoffStore) GetPackage(packageID string) (*autonomy.HandoffPackage, error) {
	var pkg *autonomy.HandoffPackage
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketHandoffPackages)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(packageID))
		if v == nil {
			return fmt.Errorf("package %q not found", packageID)
		}
		var p autonomy.HandoffPackage
		if err := jsonUnmarshal(v, &p); err != nil {
			return err
		}
		pkg = copyHandoffPackage(&p)
		return nil
	})
	return pkg, err
}

func (s *HandoffStore) ListPackages() ([]*autonomy.HandoffPackage, error) {
	var result []*autonomy.HandoffPackage
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketHandoffPackages)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var p autonomy.HandoffPackage
			if err := jsonUnmarshal(v, &p); err != nil {
				return err
			}
			result = append(result, copyHandoffPackage(&p))
			return nil
		})
	})
	return result, err
}

func (s *HandoffStore) GetByTaskID(taskID string) (*autonomy.HandoffPackage, error) {
	var packageID []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		idx := tx.Bucket(bucketHandoffsByTaskID)
		if idx == nil {
			return ErrBucketNotFound
		}
		packageID = idx.Get([]byte(taskID))
		return nil
	})
	if err != nil || packageID == nil {
		return nil, fmt.Errorf("no package for task %q", taskID)
	}
	return s.GetPackage(string(packageID))
}
