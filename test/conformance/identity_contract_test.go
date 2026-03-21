package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/identity"
)

func TestIdentityContract_JSONFieldNames_SnakeCase(t *testing.T) {
	id := &identity.AppIdentity{
		IdentityID:     "id_contract",
		IdentityType:   identity.IdentityTypeGitHubApp,
		AppID:          "123",
		InstallationID: "456",
		OrgScope:       "org1",
		RepoScope:      "repo1",
		Capabilities:   []identity.Capability{identity.CapReadRepo},
		CreatedAt:      time.Now().UTC(),
	}

	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{
		"identity_id", "identity_type", "app_id", "installation_id",
		"org_scope", "repo_scope", "capabilities", "created_at",
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, "\""+field+"\"") {
			t.Errorf("JSON missing snake_case field %q", field)
		}
	}
}

func TestIdentityContract_ScopeGrantSnakeCase(t *testing.T) {
	g := identity.ScopeGrant{
		ScopeType:    identity.ScopeRepository,
		ScopeValue:   "owner/repo",
		Capabilities: []identity.Capability{identity.CapReadRepo},
	}

	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{"scope_type", "scope_value", "capabilities"}
	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, "\""+field+"\"") {
			t.Errorf("ScopeGrant JSON missing snake_case field %q", field)
		}
	}
}

func TestIdentityContract_IdentityTypeValues(t *testing.T) {
	types := []identity.IdentityType{
		identity.IdentityTypeGitHubApp,
		identity.IdentityTypePAT,
		identity.IdentityTypeToken,
	}

	for _, tt := range types {
		if strings.ToLower(string(tt)) != string(tt) {
			t.Errorf("identity type %q should be lower_snake_case", tt)
		}
	}
}

func TestIdentityContract_CapabilityValues(t *testing.T) {
	caps := []identity.Capability{
		identity.CapReadRepo, identity.CapWriteRepo,
		identity.CapReadIssues, identity.CapWriteIssues,
		identity.CapReadPRs, identity.CapWritePRs, identity.CapAdmin,
	}

	for _, c := range caps {
		if strings.ToLower(string(c)) != string(c) {
			t.Errorf("capability %q should be lower_snake_case", c)
		}
	}
}

func TestIdentityContract_AppIdentity_JSONRoundTrip(t *testing.T) {
	orig := &identity.AppIdentity{
		IdentityID:     "id_roundtrip",
		IdentityType:   identity.IdentityTypeGitHubApp,
		AppID:          "123",
		InstallationID: "456",
		OrgScope:       "my-org",
		RepoScope:      "my-org/repo",
		Capabilities:   []identity.Capability{identity.CapReadRepo, identity.CapWritePRs},
		ScopeGrants:    []identity.ScopeGrant{{ScopeType: identity.ScopeRepository, ScopeValue: "owner/repo", Capabilities: []identity.Capability{identity.CapReadRepo}}},
		CreatedAt:      time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded identity.AppIdentity
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.IdentityID != orig.IdentityID {
		t.Errorf("IdentityID: got %q, want %q", decoded.IdentityID, orig.IdentityID)
	}
	if decoded.IdentityType != orig.IdentityType {
		t.Errorf("IdentityType: got %q, want %q", decoded.IdentityType, orig.IdentityType)
	}
	if decoded.AppID != orig.AppID {
		t.Errorf("AppID: got %q, want %q", decoded.AppID, orig.AppID)
	}
	if decoded.InstallationID != orig.InstallationID {
		t.Errorf("InstallationID: got %q, want %q", decoded.InstallationID, orig.InstallationID)
	}
	if decoded.OrgScope != orig.OrgScope {
		t.Errorf("OrgScope: got %q, want %q", decoded.OrgScope, orig.OrgScope)
	}
	if decoded.RepoScope != orig.RepoScope {
		t.Errorf("RepoScope: got %q, want %q", decoded.RepoScope, orig.RepoScope)
	}
	if len(decoded.Capabilities) != len(orig.Capabilities) {
		t.Errorf("Capabilities len: got %d, want %d", len(decoded.Capabilities), len(orig.Capabilities))
	}
	if len(decoded.ScopeGrants) != len(orig.ScopeGrants) {
		t.Errorf("ScopeGrants len: got %d, want %d", len(decoded.ScopeGrants), len(orig.ScopeGrants))
	}
	if !decoded.CreatedAt.Equal(orig.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", decoded.CreatedAt, orig.CreatedAt)
	}
}

func TestIdentityContract_ScopeGrant_JSONRoundTrip(t *testing.T) {
	orig := identity.ScopeGrant{
		ScopeType:    identity.ScopeRepository,
		ScopeValue:   "owner/repo",
		Capabilities: []identity.Capability{identity.CapReadRepo, identity.CapWritePRs},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded identity.ScopeGrant
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ScopeType != orig.ScopeType {
		t.Errorf("ScopeType: got %q, want %q", decoded.ScopeType, orig.ScopeType)
	}
	if decoded.ScopeValue != orig.ScopeValue {
		t.Errorf("ScopeValue: got %q, want %q", decoded.ScopeValue, orig.ScopeValue)
	}
	if len(decoded.Capabilities) != len(orig.Capabilities) {
		t.Errorf("Capabilities len: got %d, want %d", len(decoded.Capabilities), len(orig.Capabilities))
	}
}
