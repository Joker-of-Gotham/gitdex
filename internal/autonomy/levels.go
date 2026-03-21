package autonomy

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AutonomyLevel string

const (
	LevelManual     AutonomyLevel = "manual"
	LevelSupervised AutonomyLevel = "supervised"
	LevelAutonomous AutonomyLevel = "autonomous"
	LevelFullAuto   AutonomyLevel = "full_auto"
)

type CapabilityAutonomy struct {
	Capability       string        `json:"capability" yaml:"capability"`
	Level            AutonomyLevel `json:"level" yaml:"level"`
	Constraints      []string      `json:"constraints" yaml:"constraints"`
	RequiresApproval bool          `json:"requires_approval" yaml:"requires_approval"`
}

type AutonomyConfig struct {
	ConfigID             string               `json:"config_id" yaml:"config_id"`
	Name                 string               `json:"name" yaml:"name"`
	CapabilityAutonomies []CapabilityAutonomy `json:"capability_autonomies" yaml:"capability_autonomies"`
	DefaultLevel         AutonomyLevel        `json:"default_level" yaml:"default_level"`
	CreatedAt            time.Time            `json:"created_at" yaml:"created_at"`
}

type AutonomyStore interface {
	SaveConfig(cfg *AutonomyConfig) error
	GetConfig(configID string) (*AutonomyConfig, error)
	GetActiveConfig() (*AutonomyConfig, error)
	SetActiveConfig(configID string) error
	ListConfigs() ([]*AutonomyConfig, error)
}

type MemoryAutonomyStore struct {
	mu       sync.RWMutex
	configs  map[string]*AutonomyConfig
	activeID string
}

func NewMemoryAutonomyStore() *MemoryAutonomyStore {
	return &MemoryAutonomyStore{
		configs: make(map[string]*AutonomyConfig),
	}
}

func (s *MemoryAutonomyStore) SaveConfig(cfg *AutonomyConfig) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil config")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := copyAutonomyConfig(cfg)
	if cp.ConfigID == "" {
		cp.ConfigID = "cfg_" + uuid.New().String()[:8]
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now().UTC()
	}
	s.configs[cp.ConfigID] = cp
	if s.activeID == "" {
		s.activeID = cp.ConfigID
	}
	// Write back generated fields to caller's struct (inside lock, after store)
	cfg.ConfigID = cp.ConfigID
	cfg.CreatedAt = cp.CreatedAt
	return nil
}

func (s *MemoryAutonomyStore) GetConfig(configID string) (*AutonomyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.configs[configID]
	if !ok {
		return nil, fmt.Errorf("config %q not found", configID)
	}
	return copyAutonomyConfig(c), nil
}

func (s *MemoryAutonomyStore) GetActiveConfig() (*AutonomyConfig, error) {
	s.mu.RLock()
	activeID := s.activeID
	s.mu.RUnlock()

	if activeID == "" {
		return nil, nil
	}
	return s.GetConfig(activeID)
}

func (s *MemoryAutonomyStore) SetActiveConfig(configID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.configs[configID]; !ok {
		return fmt.Errorf("config %q not found", configID)
	}
	s.activeID = configID
	return nil
}

func (s *MemoryAutonomyStore) ListConfigs() ([]*AutonomyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*AutonomyConfig, 0, len(s.configs))
	for _, c := range s.configs {
		result = append(result, copyAutonomyConfig(c))
	}
	return result, nil
}

func copyAutonomyConfig(cfg *AutonomyConfig) *AutonomyConfig {
	if cfg == nil {
		return nil
	}
	cp := *cfg
	if len(cfg.CapabilityAutonomies) > 0 {
		cp.CapabilityAutonomies = make([]CapabilityAutonomy, len(cfg.CapabilityAutonomies))
		for i := range cfg.CapabilityAutonomies {
			cp.CapabilityAutonomies[i] = cfg.CapabilityAutonomies[i]
			if len(cfg.CapabilityAutonomies[i].Constraints) > 0 {
				cp.CapabilityAutonomies[i].Constraints = make([]string, len(cfg.CapabilityAutonomies[i].Constraints))
				copy(cp.CapabilityAutonomies[i].Constraints, cfg.CapabilityAutonomies[i].Constraints)
			}
		}
	}
	return &cp
}
