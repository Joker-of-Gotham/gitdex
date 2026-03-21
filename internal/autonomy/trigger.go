package autonomy

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TriggerType string

const (
	TriggerTypeEvent TriggerType = "event"
	TriggerSchedule  TriggerType = "schedule"
	TriggerAPI       TriggerType = "api"
	TriggerOperator  TriggerType = "operator"
)

type TriggerConfig struct {
	TriggerID      string      `json:"trigger_id" yaml:"trigger_id"`
	TriggerType    TriggerType `json:"trigger_type" yaml:"trigger_type"`
	Name           string      `json:"name" yaml:"name"`
	Source         string      `json:"source" yaml:"source"`
	Pattern        string      `json:"pattern" yaml:"pattern"`
	ActionTemplate string      `json:"action_template" yaml:"action_template"`
	Enabled        bool        `json:"enabled" yaml:"enabled"`
	CreatedAt      time.Time   `json:"created_at" yaml:"created_at"`
}

type TriggerEvent struct {
	EventID         string      `json:"event_id" yaml:"event_id"`
	TriggerID       string      `json:"trigger_id" yaml:"trigger_id"`
	TriggerType     TriggerType `json:"trigger_type" yaml:"trigger_type"`
	SourceEvent     string      `json:"source_event" yaml:"source_event"`
	ResultingTaskID string      `json:"resulting_task_id" yaml:"resulting_task_id"`
	Timestamp       time.Time   `json:"timestamp" yaml:"timestamp"`
}

type TriggerStore interface {
	SaveTrigger(cfg *TriggerConfig) error
	GetTrigger(triggerID string) (*TriggerConfig, error)
	ListTriggers() ([]*TriggerConfig, error)
	EnableTrigger(triggerID string) error
	DisableTrigger(triggerID string) error
	AppendTriggerEvent(ev *TriggerEvent) error
	ListTriggerEvents(triggerID string, limit int) ([]*TriggerEvent, error)
}

type MemoryTriggerStore struct {
	mu      sync.RWMutex
	configs map[string]*TriggerConfig
	events  []*TriggerEvent
}

func NewMemoryTriggerStore() *MemoryTriggerStore {
	return &MemoryTriggerStore{
		configs: make(map[string]*TriggerConfig),
		events:  make([]*TriggerEvent, 0),
	}
}

func (s *MemoryTriggerStore) SaveTrigger(cfg *TriggerConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil trigger config")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *cfg
	if cp.TriggerID == "" {
		cp.TriggerID = "tr_" + uuid.New().String()[:8]
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now().UTC()
	}
	s.configs[cp.TriggerID] = &cp
	cfg.TriggerID = cp.TriggerID
	cfg.CreatedAt = cp.CreatedAt
	return nil
}

func (s *MemoryTriggerStore) GetTrigger(triggerID string) (*TriggerConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, ok := s.configs[triggerID]
	if !ok {
		return nil, fmt.Errorf("trigger %q not found", triggerID)
	}
	cp := *cfg
	return &cp, nil
}

func (s *MemoryTriggerStore) ListTriggers() ([]*TriggerConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*TriggerConfig, 0, len(s.configs))
	for _, cfg := range s.configs {
		cp := *cfg
		result = append(result, &cp)
	}
	return result, nil
}

func (s *MemoryTriggerStore) EnableTrigger(triggerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := s.configs[triggerID]
	if !ok {
		return fmt.Errorf("trigger %q not found", triggerID)
	}
	cfg.Enabled = true
	return nil
}

func (s *MemoryTriggerStore) DisableTrigger(triggerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := s.configs[triggerID]
	if !ok {
		return fmt.Errorf("trigger %q not found", triggerID)
	}
	cfg.Enabled = false
	return nil
}

func (s *MemoryTriggerStore) AppendTriggerEvent(ev *TriggerEvent) error {
	if ev == nil {
		return fmt.Errorf("cannot append nil trigger event")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *ev
	if cp.EventID == "" {
		cp.EventID = "tev_" + uuid.New().String()[:8]
	}
	if cp.Timestamp.IsZero() {
		cp.Timestamp = time.Now().UTC()
	}
	s.events = append(s.events, &cp)
	ev.EventID = cp.EventID
	ev.Timestamp = cp.Timestamp
	return nil
}

func (s *MemoryTriggerStore) ListTriggerEvents(triggerID string, limit int) ([]*TriggerEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var result []*TriggerEvent
	for i := len(s.events) - 1; i >= 0; i-- {
		ev := s.events[i]
		if triggerID != "" && ev.TriggerID != triggerID {
			continue
		}
		cp := *ev
		result = append(result, &cp)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}
