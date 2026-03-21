package policy

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PolicyBundle struct {
	BundleID          string             `json:"bundle_id" yaml:"bundle_id"`
	Name              string             `json:"name" yaml:"name"`
	Version           string             `json:"version" yaml:"version"`
	CapabilityGrants  []CapabilityGrant  `json:"capability_grants" yaml:"capability_grants"`
	ProtectedTargets  []ProtectedTarget  `json:"protected_targets" yaml:"protected_targets"`
	ApprovalRules     []ApprovalRule     `json:"approval_rules" yaml:"approval_rules"`
	RiskThresholds    map[string]string  `json:"risk_thresholds,omitempty" yaml:"risk_thresholds,omitempty"`
	DataHandlingRules []DataHandlingRule `json:"data_handling_rules,omitempty" yaml:"data_handling_rules,omitempty"`
	CreatedAt         time.Time          `json:"created_at" yaml:"created_at"`
}

type CapabilityGrant struct {
	Scope        string   `json:"scope" yaml:"scope"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
	Conditions   []string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

type TargetType string

const (
	TargetBranch      TargetType = "branch"
	TargetEnvironment TargetType = "environment"
	TargetPath        TargetType = "path"
)

type ProtectedTarget struct {
	TargetType      TargetType `json:"target_type" yaml:"target_type"`
	Pattern         string     `json:"pattern" yaml:"pattern"`
	ProtectionLevel string     `json:"protection_level" yaml:"protection_level"`
}

type ApprovalType string

const (
	ApprovalOwner    ApprovalType = "owner"
	ApprovalSecurity ApprovalType = "security"
	ApprovalRelease  ApprovalType = "release"
	ApprovalQuorum   ApprovalType = "quorum"
)

type ApprovalRule struct {
	ActionPattern     string       `json:"action_pattern" yaml:"action_pattern"`
	RequiredApprovers []string     `json:"required_approvers" yaml:"required_approvers"`
	ApprovalType      ApprovalType `json:"approval_type" yaml:"approval_type"`
}

type DataHandlingRule struct {
	RuleType string `json:"rule_type" yaml:"rule_type"`
	Pattern  string `json:"pattern" yaml:"pattern"`
	Action   string `json:"action" yaml:"action"`
}

type PolicyBundleStore interface {
	SaveBundle(bundle *PolicyBundle) error
	GetBundle(bundleID string) (*PolicyBundle, error)
	ListBundles() ([]*PolicyBundle, error)
	GetActiveBundle() (*PolicyBundle, error)
	SetActiveBundle(bundleID string) error
}

type MemoryBundleStore struct {
	mu       sync.RWMutex
	bundles  map[string]*PolicyBundle
	activeID string
}

func NewMemoryBundleStore() *MemoryBundleStore {
	return &MemoryBundleStore{
		bundles: make(map[string]*PolicyBundle),
	}
}

func deepCopyBundle(b *PolicyBundle) *PolicyBundle {
	cp := *b
	if len(b.CapabilityGrants) > 0 {
		cp.CapabilityGrants = make([]CapabilityGrant, len(b.CapabilityGrants))
		for i, g := range b.CapabilityGrants {
			cp.CapabilityGrants[i] = CapabilityGrant{
				Scope:        g.Scope,
				Capabilities: append([]string(nil), g.Capabilities...),
				Conditions:   append([]string(nil), g.Conditions...),
			}
		}
	}
	if len(b.ProtectedTargets) > 0 {
		cp.ProtectedTargets = make([]ProtectedTarget, len(b.ProtectedTargets))
		copy(cp.ProtectedTargets, b.ProtectedTargets)
	}
	if len(b.ApprovalRules) > 0 {
		cp.ApprovalRules = make([]ApprovalRule, len(b.ApprovalRules))
		for i, r := range b.ApprovalRules {
			cp.ApprovalRules[i] = ApprovalRule{
				ActionPattern:     r.ActionPattern,
				RequiredApprovers: append([]string(nil), r.RequiredApprovers...),
				ApprovalType:      r.ApprovalType,
			}
		}
	}
	if b.RiskThresholds != nil {
		cp.RiskThresholds = make(map[string]string, len(b.RiskThresholds))
		for k, v := range b.RiskThresholds {
			cp.RiskThresholds[k] = v
		}
	}
	if len(b.DataHandlingRules) > 0 {
		cp.DataHandlingRules = make([]DataHandlingRule, len(b.DataHandlingRules))
		copy(cp.DataHandlingRules, b.DataHandlingRules)
	}
	return &cp
}

func (s *MemoryBundleStore) SaveBundle(bundle *PolicyBundle) error {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := deepCopyBundle(bundle)
	s.bundles[bundle.BundleID] = cp
	if s.activeID == "" {
		s.activeID = bundle.BundleID
	}
	return nil
}

func (s *MemoryBundleStore) GetBundle(bundleID string) (*PolicyBundle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.bundles[bundleID]
	if !ok {
		return nil, fmt.Errorf("bundle %q not found", bundleID)
	}
	return deepCopyBundle(b), nil
}

func (s *MemoryBundleStore) ListBundles() ([]*PolicyBundle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*PolicyBundle, 0, len(s.bundles))
	for _, b := range s.bundles {
		result = append(result, deepCopyBundle(b))
	}
	return result, nil
}

func (s *MemoryBundleStore) GetActiveBundle() (*PolicyBundle, error) {
	s.mu.RLock()
	activeID := s.activeID
	s.mu.RUnlock()

	if activeID == "" {
		return nil, nil
	}
	return s.GetBundle(activeID)
}

func (s *MemoryBundleStore) SetActiveBundle(bundleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.bundles[bundleID]; !ok {
		return fmt.Errorf("bundle %q not found", bundleID)
	}
	s.activeID = bundleID
	return nil
}
