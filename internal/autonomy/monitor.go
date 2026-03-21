package autonomy

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MonitorConfig represents a monitored repository configuration.
type MonitorConfig struct {
	MonitorID string   `json:"monitor_id" yaml:"monitor_id"`
	RepoOwner string   `json:"repo_owner" yaml:"repo_owner"`
	RepoName  string   `json:"repo_name" yaml:"repo_name"`
	Interval  string   `json:"interval" yaml:"interval"`
	Checks    []string `json:"checks" yaml:"checks"`
	Enabled   bool     `json:"enabled" yaml:"enabled"`
}

// MonitorEvent represents an event emitted by a monitor check.
type MonitorEvent struct {
	EventID   string    `json:"event_id" yaml:"event_id"`
	MonitorID string    `json:"monitor_id" yaml:"monitor_id"`
	RepoOwner string    `json:"repo_owner" yaml:"repo_owner"`
	RepoName  string    `json:"repo_name" yaml:"repo_name"`
	CheckName string    `json:"check_name" yaml:"check_name"`
	Status    string    `json:"status" yaml:"status"` // ok, warning, critical
	Message   string    `json:"message" yaml:"message"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// MonitorEventFilter filters events for ListEvents.
type MonitorEventFilter struct {
	MonitorID string
	RepoOwner string
	RepoName  string
	Limit     int
}

// MonitorStore defines persistence for monitor configs and events.
type MonitorStore interface {
	SaveMonitorConfig(cfg *MonitorConfig) error
	GetMonitorConfig(monitorID string) (*MonitorConfig, error)
	ListMonitorConfigs() ([]*MonitorConfig, error)
	AppendEvent(ev *MonitorEvent) error
	ListEvents(filter MonitorEventFilter) ([]*MonitorEvent, error)
	RemoveMonitorConfig(monitorID string) error
}

// MemoryMonitorStore is a thread-safe in-memory implementation of MonitorStore.
type MemoryMonitorStore struct {
	mu      sync.RWMutex
	configs map[string]*MonitorConfig
	events  []*MonitorEvent
}

// NewMemoryMonitorStore returns a new MemoryMonitorStore.
func NewMemoryMonitorStore() *MemoryMonitorStore {
	return &MemoryMonitorStore{
		configs: make(map[string]*MonitorConfig),
		events:  make([]*MonitorEvent, 0),
	}
}

func (s *MemoryMonitorStore) SaveMonitorConfig(cfg *MonitorConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil monitor config")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *cfg
	if len(cfg.Checks) > 0 {
		cp.Checks = make([]string, len(cfg.Checks))
		copy(cp.Checks, cfg.Checks)
	}
	if cp.MonitorID == "" {
		cp.MonitorID = "mon_" + uuid.New().String()[:8]
	}
	s.configs[cp.MonitorID] = &cp
	cfg.MonitorID = cp.MonitorID
	return nil
}

func (s *MemoryMonitorStore) GetMonitorConfig(monitorID string) (*MonitorConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, ok := s.configs[monitorID]
	if !ok {
		return nil, fmt.Errorf("monitor %q not found", monitorID)
	}
	cp := *cfg
	if len(cfg.Checks) > 0 {
		cp.Checks = make([]string, len(cfg.Checks))
		copy(cp.Checks, cfg.Checks)
	}
	return &cp, nil
}

func (s *MemoryMonitorStore) ListMonitorConfigs() ([]*MonitorConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*MonitorConfig, 0, len(s.configs))
	for _, cfg := range s.configs {
		cp := *cfg
		if len(cfg.Checks) > 0 {
			cp.Checks = make([]string, len(cfg.Checks))
			copy(cp.Checks, cfg.Checks)
		}
		result = append(result, &cp)
	}
	return result, nil
}

func (s *MemoryMonitorStore) AppendEvent(ev *MonitorEvent) error {
	if ev == nil {
		return fmt.Errorf("cannot append nil event")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *ev
	if cp.EventID == "" {
		cp.EventID = "ev_" + uuid.New().String()[:8]
	}
	if cp.Timestamp.IsZero() {
		cp.Timestamp = time.Now().UTC()
	}
	s.events = append(s.events, &cp)
	ev.EventID = cp.EventID
	ev.Timestamp = cp.Timestamp
	return nil
}

func (s *MemoryMonitorStore) ListEvents(filter MonitorEventFilter) ([]*MonitorEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	var result []*MonitorEvent
	for i := len(s.events) - 1; i >= 0; i-- {
		ev := s.events[i]
		if filter.MonitorID != "" && ev.MonitorID != filter.MonitorID {
			continue
		}
		if filter.RepoOwner != "" && ev.RepoOwner != filter.RepoOwner {
			continue
		}
		if filter.RepoName != "" && ev.RepoName != filter.RepoName {
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

func (s *MemoryMonitorStore) RemoveMonitorConfig(monitorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.configs[monitorID]; !ok {
		return fmt.Errorf("monitor %q not found", monitorID)
	}
	delete(s.configs, monitorID)
	return nil
}
