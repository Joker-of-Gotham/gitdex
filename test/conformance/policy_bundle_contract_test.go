package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/policy"
)

func TestPolicyBundleContract_JSONFieldNames_SnakeCase(t *testing.T) {
	b := &policy.PolicyBundle{
		BundleID:  "bundle_contract",
		Name:      "test",
		Version:   "1.0.0",
		CreatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{
		"bundle_id", "name", "version", "capability_grants",
		"protected_targets", "approval_rules", "created_at",
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, "\""+field+"\"") {
			t.Errorf("JSON missing snake_case field %q", field)
		}
	}
}

func TestPolicyBundleContract_CapabilityGrantSnakeCase(t *testing.T) {
	g := policy.CapabilityGrant{
		Scope:        "repo",
		Capabilities: []string{"read_repo"},
		Conditions:   []string{"env=prod"},
	}

	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{"scope", "capabilities", "conditions"}
	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, "\""+field+"\"") {
			t.Errorf("CapabilityGrant JSON missing snake_case field %q", field)
		}
	}
}

func TestPolicyBundleContract_ProtectedTargetSnakeCase(t *testing.T) {
	pt := policy.ProtectedTarget{
		TargetType:      policy.TargetBranch,
		Pattern:         "main",
		ProtectionLevel: "strict",
	}

	data, err := json.Marshal(pt)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{"target_type", "pattern", "protection_level"}
	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, "\""+field+"\"") {
			t.Errorf("ProtectedTarget JSON missing snake_case field %q", field)
		}
	}
}

func TestPolicyBundleContract_ApprovalTypeValues(t *testing.T) {
	types := []policy.ApprovalType{
		policy.ApprovalOwner,
		policy.ApprovalSecurity,
		policy.ApprovalRelease,
		policy.ApprovalQuorum,
	}

	for _, tt := range types {
		if strings.ToLower(string(tt)) != string(tt) {
			t.Errorf("approval type %q should be lower_snake_case", tt)
		}
	}
}

func TestPolicyBundleContract_TimestampRFC3339(t *testing.T) {
	b := &policy.PolicyBundle{
		BundleID:  "b_ts",
		Name:      "ts",
		CreatedAt: time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if !strings.Contains(string(data), "2026-03-19T12:00:00Z") {
		t.Errorf("timestamp not in RFC3339 format: %s", string(data))
	}
}

func TestPolicyBundleContract_PolicyBundle_JSONRoundTrip(t *testing.T) {
	orig := &policy.PolicyBundle{
		BundleID:         "bundle_roundtrip",
		Name:             "my-policy",
		Version:          "1.0.0",
		CapabilityGrants: []policy.CapabilityGrant{{Scope: "repo", Capabilities: []string{"read_repo"}, Conditions: []string{"env=prod"}}},
		ProtectedTargets: []policy.ProtectedTarget{{TargetType: policy.TargetBranch, Pattern: "main", ProtectionLevel: "strict"}},
		ApprovalRules:    []policy.ApprovalRule{{ActionPattern: "merge", RequiredApprovers: []string{"alice"}, ApprovalType: policy.ApprovalOwner}},
		CreatedAt:        time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded policy.PolicyBundle
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.BundleID != orig.BundleID || decoded.Name != orig.Name || decoded.Version != orig.Version {
		t.Errorf("BundleID/Name/Version: got %q/%q/%q, want %q/%q/%q", decoded.BundleID, decoded.Name, decoded.Version, orig.BundleID, orig.Name, orig.Version)
	}
	if len(decoded.CapabilityGrants) != 1 || decoded.CapabilityGrants[0].Scope != "repo" {
		t.Errorf("CapabilityGrants: got %v", decoded.CapabilityGrants)
	}
	if len(decoded.ProtectedTargets) != 1 || decoded.ProtectedTargets[0].Pattern != "main" {
		t.Errorf("ProtectedTargets: got %v", decoded.ProtectedTargets)
	}
	if len(decoded.ApprovalRules) != 1 || decoded.ApprovalRules[0].ActionPattern != "merge" {
		t.Errorf("ApprovalRules: got %v", decoded.ApprovalRules)
	}
	if !decoded.CreatedAt.Equal(orig.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", decoded.CreatedAt, orig.CreatedAt)
	}
}

func TestPolicyBundleContract_CapabilityGrant_JSONRoundTrip(t *testing.T) {
	orig := policy.CapabilityGrant{
		Scope:        "scope/repo",
		Capabilities: []string{"read_repo", "write_prs"},
		Conditions:   []string{"env=prod", "team=eng"},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded policy.CapabilityGrant
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Scope != orig.Scope || len(decoded.Capabilities) != len(orig.Capabilities) || len(decoded.Conditions) != len(orig.Conditions) {
		t.Errorf("CapabilityGrant round-trip: got %+v, want %+v", decoded, orig)
	}
}

func TestPolicyBundleContract_ProtectedTarget_JSONRoundTrip(t *testing.T) {
	orig := policy.ProtectedTarget{
		TargetType:      policy.TargetBranch,
		Pattern:         "main",
		ProtectionLevel: "strict",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded policy.ProtectedTarget
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.TargetType != orig.TargetType || decoded.Pattern != orig.Pattern || decoded.ProtectionLevel != orig.ProtectionLevel {
		t.Errorf("ProtectedTarget round-trip: got %+v, want %+v", decoded, orig)
	}
}

func TestPolicyBundleContract_ApprovalRule_JSONRoundTrip(t *testing.T) {
	orig := policy.ApprovalRule{
		ActionPattern:     "merge",
		RequiredApprovers: []string{"alice", "bob"},
		ApprovalType:      policy.ApprovalSecurity,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded policy.ApprovalRule
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ActionPattern != orig.ActionPattern || decoded.ApprovalType != orig.ApprovalType || len(decoded.RequiredApprovers) != 2 {
		t.Errorf("ApprovalRule round-trip: got %+v, want %+v", decoded, orig)
	}
}
