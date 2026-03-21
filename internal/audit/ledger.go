package audit

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

type EventType string

const (
	EventPlanCreated        EventType = "plan_created"
	EventPlanApproved       EventType = "plan_approved"
	EventPlanRejected       EventType = "plan_rejected"
	EventTaskStarted        EventType = "task_started"
	EventTaskSucceeded      EventType = "task_succeeded"
	EventTaskFailed         EventType = "task_failed"
	EventPolicyEvaluated    EventType = "policy_evaluated"
	EventEmergencyControl   EventType = "emergency_control"
	EventIdentityRegistered EventType = "identity_registered"
	EventRecovery           EventType = "recovery"
	EventTaskControl        EventType = "task_control"
)

type AuditEntry struct {
	EntryID       string    `json:"entry_id" yaml:"entry_id"`
	CorrelationID string    `json:"correlation_id" yaml:"correlation_id"`
	TaskID        string    `json:"task_id" yaml:"task_id"`
	PlanID        string    `json:"plan_id" yaml:"plan_id"`
	EventType     EventType `json:"event_type" yaml:"event_type"`
	Actor         string    `json:"actor" yaml:"actor"`
	Action        string    `json:"action" yaml:"action"`
	Target        string    `json:"target" yaml:"target"`
	PolicyResult  string    `json:"policy_result,omitempty" yaml:"policy_result,omitempty"`
	EvidenceRefs  []string  `json:"evidence_refs,omitempty" yaml:"evidence_refs,omitempty"`
	Timestamp     time.Time `json:"timestamp" yaml:"timestamp"`
}

type AuditFilter struct {
	EntryID       string
	EventType     EventType
	TaskID        string
	CorrelationID string
	FromTime      *time.Time
	ToTime        *time.Time
}

type AuditLedger interface {
	Append(entry *AuditEntry) error
	Query(filters AuditFilter) ([]*AuditEntry, error)
	GetByCorrelation(correlationID string) ([]*AuditEntry, error)
	GetByTask(taskID string) ([]*AuditEntry, error)
	GetByEntryID(entryID string) (*AuditEntry, bool)
}

type MemoryAuditLedger struct {
	mu      sync.RWMutex
	entries []*AuditEntry
	byID    map[string]*AuditEntry
}

func NewMemoryAuditLedger() *MemoryAuditLedger {
	return &MemoryAuditLedger{
		entries: nil,
		byID:    make(map[string]*AuditEntry),
	}
}

func GenerateEntryID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("audit_%x", b)
}

func (l *MemoryAuditLedger) Append(entry *AuditEntry) error {
	if entry == nil {
		return fmt.Errorf("cannot append nil entry")
	}
	if entry.EntryID == "" {
		entry.EntryID = GenerateEntryID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	cp := *entry
	cp.EvidenceRefs = make([]string, len(entry.EvidenceRefs))
	copy(cp.EvidenceRefs, entry.EvidenceRefs)
	l.entries = append(l.entries, &cp)
	l.byID[entry.EntryID] = &cp
	return nil
}

func (l *MemoryAuditLedger) Query(filters AuditFilter) ([]*AuditEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var out []*AuditEntry
	for _, e := range l.entries {
		if filters.EntryID != "" && e.EntryID != filters.EntryID {
			continue
		}
		if filters.EventType != "" && e.EventType != filters.EventType {
			continue
		}
		if filters.TaskID != "" && e.TaskID != filters.TaskID {
			continue
		}
		if filters.CorrelationID != "" && e.CorrelationID != filters.CorrelationID {
			continue
		}
		if filters.FromTime != nil && e.Timestamp.Before(*filters.FromTime) {
			continue
		}
		if filters.ToTime != nil && e.Timestamp.After(*filters.ToTime) {
			continue
		}
		cp := *e
		cp.EvidenceRefs = make([]string, len(e.EvidenceRefs))
		copy(cp.EvidenceRefs, e.EvidenceRefs)
		out = append(out, &cp)
	}
	return out, nil
}

func (l *MemoryAuditLedger) GetByCorrelation(correlationID string) ([]*AuditEntry, error) {
	return l.Query(AuditFilter{CorrelationID: correlationID})
}

func (l *MemoryAuditLedger) GetByTask(taskID string) ([]*AuditEntry, error) {
	return l.Query(AuditFilter{TaskID: taskID})
}

func (l *MemoryAuditLedger) GetByEntryID(entryID string) (*AuditEntry, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	e, ok := l.byID[entryID]
	if !ok {
		return nil, false
	}
	cp := *e
	cp.EvidenceRefs = make([]string, len(e.EvidenceRefs))
	copy(cp.EvidenceRefs, e.EvidenceRefs)
	return &cp, true
}
