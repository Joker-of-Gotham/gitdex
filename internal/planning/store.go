package planning

import (
	"fmt"
	"sync"
	"time"
)

type PlanStore interface {
	Save(plan *Plan) error
	Get(planID string) (*Plan, error)
	GetByTaskID(taskID string) (*Plan, error)
	List() ([]*Plan, error)
	UpdateStatus(planID string, status PlanStatus) error
	SaveApproval(record *ApprovalRecord) error
	GetApprovals(planID string) ([]*ApprovalRecord, error)
}

type MemoryPlanStore struct {
	mu        sync.RWMutex
	plans     map[string]*Plan
	approvals map[string][]*ApprovalRecord
}

func NewMemoryPlanStore() *MemoryPlanStore {
	return &MemoryPlanStore{
		plans:     make(map[string]*Plan),
		approvals: make(map[string][]*ApprovalRecord),
	}
}

func (s *MemoryPlanStore) Save(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("cannot save nil plan")
	}
	if plan.PlanID == "" {
		plan.PlanID = GeneratePlanID()
	}
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = time.Now().UTC()
	}
	plan.UpdatedAt = time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *plan
	s.plans[plan.PlanID] = &cp
	return nil
}

func (s *MemoryPlanStore) Get(planID string) (*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.plans[planID]
	if !ok {
		return nil, fmt.Errorf("plan %q not found", planID)
	}
	cp := *p
	return &cp, nil
}

func (s *MemoryPlanStore) GetByTaskID(taskID string) (*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.plans {
		if p.TaskID == taskID {
			cp := *p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("no plan found for task %q", taskID)
}

func (s *MemoryPlanStore) List() ([]*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Plan, 0, len(s.plans))
	for _, p := range s.plans {
		cp := *p
		result = append(result, &cp)
	}
	return result, nil
}

func (s *MemoryPlanStore) UpdateStatus(planID string, status PlanStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.plans[planID]
	if !ok {
		return fmt.Errorf("plan %q not found", planID)
	}
	p.Status = status
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *MemoryPlanStore) SaveApproval(record *ApprovalRecord) error {
	if record == nil {
		return fmt.Errorf("cannot save nil approval record")
	}
	if record.RecordID == "" {
		record.RecordID = GenerateApprovalID()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *record
	s.approvals[record.PlanID] = append(s.approvals[record.PlanID], &cp)
	return nil
}

func (s *MemoryPlanStore) GetApprovals(planID string) ([]*ApprovalRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.approvals[planID]
	result := make([]*ApprovalRecord, len(records))
	for i, r := range records {
		cp := *r
		result[i] = &cp
	}
	return result, nil
}
