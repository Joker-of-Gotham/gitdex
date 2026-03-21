package bbolt

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/your-org/gitdex/internal/planning"
)

// PlanStore implements planning.PlanStore using BBolt.
type PlanStore struct {
	db *bbolt.DB
}

// NewPlanStore creates a new PlanStore.
func NewPlanStore(db *bbolt.DB) *PlanStore {
	return &PlanStore{db: db}
}

func (s *PlanStore) Save(plan *planning.Plan) error {
	if plan == nil {
		return fmt.Errorf("cannot save nil plan")
	}
	if plan.PlanID == "" {
		plan.PlanID = planning.GeneratePlanID()
	}
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = time.Now().UTC()
	}
	plan.UpdatedAt = time.Now().UTC()

	return s.db.Update(func(tx *bbolt.Tx) error {
		main := tx.Bucket(bucketPlans)
		idx := tx.Bucket(bucketPlansByTaskID)
		if main == nil || idx == nil {
			return ErrBucketNotFound
		}

		data, err := jsonMarshal(plan)
		if err != nil {
			return err
		}
		if err := main.Put([]byte(plan.PlanID), data); err != nil {
			return err
		}

		// Update secondary index
		if plan.TaskID != "" {
			if err := idx.Put([]byte(plan.TaskID), []byte(plan.PlanID)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PlanStore) Get(planID string) (*planning.Plan, error) {
	var plan *planning.Plan
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPlans)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(planID))
		if v == nil {
			return fmt.Errorf("plan %q not found", planID)
		}
		var p planning.Plan
		if err := jsonUnmarshal(v, &p); err != nil {
			return err
		}
		plan = &p
		return nil
	})
	return plan, err
}

func (s *PlanStore) GetByTaskID(taskID string) (*planning.Plan, error) {
	var planID []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		idx := tx.Bucket(bucketPlansByTaskID)
		if idx == nil {
			return ErrBucketNotFound
		}
		planID = idx.Get([]byte(taskID))
		return nil
	})
	if err != nil {
		return nil, err
	}
	if planID == nil {
		return nil, fmt.Errorf("no plan found for task %q", taskID)
	}
	return s.Get(string(planID))
}

func (s *PlanStore) List() ([]*planning.Plan, error) {
	var result []*planning.Plan
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPlans)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(_, v []byte) error {
			var p planning.Plan
			if err := jsonUnmarshal(v, &p); err != nil {
				return err
			}
			result = append(result, &p)
			return nil
		})
	})
	return result, err
}

func (s *PlanStore) UpdateStatus(planID string, status planning.PlanStatus) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPlans)
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(planID))
		if v == nil {
			return fmt.Errorf("plan %q not found", planID)
		}
		var p planning.Plan
		if err := jsonUnmarshal(v, &p); err != nil {
			return err
		}
		p.Status = status
		p.UpdatedAt = time.Now().UTC()
		data, err := jsonMarshal(&p)
		if err != nil {
			return err
		}
		return b.Put([]byte(planID), data)
	})
}

func (s *PlanStore) SaveApproval(record *planning.ApprovalRecord) error {
	if record == nil {
		return fmt.Errorf("cannot save nil approval record")
	}
	if record.RecordID == "" {
		record.RecordID = planning.GenerateApprovalID()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketApprovalRecords)
		if b == nil {
			return ErrBucketNotFound
		}
		data, err := jsonMarshal(record)
		if err != nil {
			return err
		}
		return b.Put([]byte(record.RecordID), data)
	})
}

func (s *PlanStore) GetApprovals(planID string) ([]*planning.ApprovalRecord, error) {
	var result []*planning.ApprovalRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketApprovalRecords)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(k, v []byte) error {
			var r planning.ApprovalRecord
			if err := jsonUnmarshal(v, &r); err != nil {
				return err
			}
			if r.PlanID == planID {
				result = append(result, &r)
			}
			return nil
		})
	})
	return result, err
}
