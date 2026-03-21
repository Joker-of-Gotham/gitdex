package identity

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type IdentityType string

const (
	IdentityTypeGitHubApp IdentityType = "github_app"
	IdentityTypePAT       IdentityType = "pat"
	IdentityTypeToken     IdentityType = "token"
)

type Capability string

const (
	CapReadRepo    Capability = "read_repo"
	CapWriteRepo   Capability = "write_repo"
	CapReadIssues  Capability = "read_issues"
	CapWriteIssues Capability = "write_issues"
	CapReadPRs     Capability = "read_prs"
	CapWritePRs    Capability = "write_prs"
	CapAdmin       Capability = "admin"
)

type ScopeType string

const (
	ScopeRepository   ScopeType = "repository"
	ScopeInstallation ScopeType = "installation"
	ScopeOrganization ScopeType = "organization"
	ScopeFleet        ScopeType = "fleet"
)

type ScopeGrant struct {
	ScopeType    ScopeType    `json:"scope_type" yaml:"scope_type"`
	ScopeValue   string       `json:"scope_value" yaml:"scope_value"`
	Capabilities []Capability `json:"capabilities" yaml:"capabilities"`
}

type AppIdentity struct {
	IdentityID     string       `json:"identity_id" yaml:"identity_id"`
	IdentityType   IdentityType `json:"identity_type" yaml:"identity_type"`
	AppID          string       `json:"app_id,omitempty" yaml:"app_id,omitempty"`
	InstallationID string       `json:"installation_id,omitempty" yaml:"installation_id,omitempty"`
	OrgScope       string       `json:"org_scope,omitempty" yaml:"org_scope,omitempty"`
	RepoScope      string       `json:"repo_scope,omitempty" yaml:"repo_scope,omitempty"`
	Capabilities   []Capability `json:"capabilities" yaml:"capabilities"`
	ScopeGrants    []ScopeGrant `json:"scope_grants,omitempty" yaml:"scope_grants,omitempty"`
	CreatedAt      time.Time    `json:"created_at" yaml:"created_at"`
}

type IdentityStore interface {
	SaveIdentity(id *AppIdentity) error
	GetIdentity(identityID string) (*AppIdentity, error)
	ListIdentities() ([]*AppIdentity, error)
	GetCurrentIdentity() (*AppIdentity, error)
	SetCurrentIdentity(identityID string) error
}

type MemoryIdentityStore struct {
	mu         sync.RWMutex
	identities map[string]*AppIdentity
	currentID  string
}

func NewMemoryIdentityStore() *MemoryIdentityStore {
	return &MemoryIdentityStore{
		identities: make(map[string]*AppIdentity),
	}
}

func (s *MemoryIdentityStore) SaveIdentity(id *AppIdentity) error {
	if id == nil {
		return fmt.Errorf("cannot save nil identity")
	}
	if id.IdentityID == "" {
		id.IdentityID = "id_" + uuid.New().String()[:8]
	}
	if id.CreatedAt.IsZero() {
		id.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *id
	if len(id.Capabilities) > 0 {
		cp.Capabilities = make([]Capability, len(id.Capabilities))
		copy(cp.Capabilities, id.Capabilities)
	}
	if len(id.ScopeGrants) > 0 {
		cp.ScopeGrants = make([]ScopeGrant, len(id.ScopeGrants))
		for i, g := range id.ScopeGrants {
			cp.ScopeGrants[i] = ScopeGrant{
				ScopeType:    g.ScopeType,
				ScopeValue:   g.ScopeValue,
				Capabilities: append([]Capability(nil), g.Capabilities...),
			}
		}
	}
	s.identities[id.IdentityID] = &cp
	if s.currentID == "" {
		s.currentID = id.IdentityID
	}
	return nil
}

func (s *MemoryIdentityStore) GetIdentity(identityID string) (*AppIdentity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.identities[identityID]
	if !ok {
		return nil, fmt.Errorf("identity %q not found", identityID)
	}
	cp := *id
	cp.Capabilities = make([]Capability, len(id.Capabilities))
	copy(cp.Capabilities, id.Capabilities)
	if len(id.ScopeGrants) > 0 {
		cp.ScopeGrants = make([]ScopeGrant, len(id.ScopeGrants))
		for i, g := range id.ScopeGrants {
			cp.ScopeGrants[i] = ScopeGrant{
				ScopeType:    g.ScopeType,
				ScopeValue:   g.ScopeValue,
				Capabilities: append([]Capability(nil), g.Capabilities...),
			}
		}
	}
	return &cp, nil
}

func (s *MemoryIdentityStore) ListIdentities() ([]*AppIdentity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*AppIdentity, 0, len(s.identities))
	for _, id := range s.identities {
		cp := *id
		cp.Capabilities = make([]Capability, len(id.Capabilities))
		copy(cp.Capabilities, id.Capabilities)
		if len(id.ScopeGrants) > 0 {
			cp.ScopeGrants = make([]ScopeGrant, len(id.ScopeGrants))
			for i, g := range id.ScopeGrants {
				cp.ScopeGrants[i] = ScopeGrant{
					ScopeType:    g.ScopeType,
					ScopeValue:   g.ScopeValue,
					Capabilities: append([]Capability(nil), g.Capabilities...),
				}
			}
		}
		result = append(result, &cp)
	}
	return result, nil
}

func (s *MemoryIdentityStore) GetCurrentIdentity() (*AppIdentity, error) {
	s.mu.RLock()
	currentID := s.currentID
	s.mu.RUnlock()

	if currentID == "" {
		return nil, nil
	}
	return s.GetIdentity(currentID)
}

func (s *MemoryIdentityStore) SetCurrentIdentity(identityID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.identities[identityID]; !ok {
		return fmt.Errorf("identity %q not found", identityID)
	}
	s.currentID = identityID
	return nil
}
