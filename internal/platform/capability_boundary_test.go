package platform

import "testing"

func TestCapabilityBoundariesIncludePartialAndInspectOnlySurfaces(t *testing.T) {
	items := CapabilityBoundaries(PlatformGitHub)
	if len(items) == 0 {
		t.Fatal("expected github capability boundaries")
	}
	found := map[string]string{}
	for _, item := range items {
		found[item.CapabilityID] = item.Mode
	}
	for capability, mode := range map[string]string{
		"dependency_graph":            "inspect_only",
		"code_scanning_tool_settings": "inspect_only",
		"notifications":               "partial_mutate",
		"email_notifications":         "partial_mutate",
		"ai_assistant_deployment":     "composed",
		"actions":                     "partial_mutate",
		"codespaces":                  "partial_mutate",
		"pull_request":                "partial_mutate",
		"dependabot_posture":          "partial_mutate",
		"secret_scanning_settings":    "partial_mutate",
		"secret_scanning_alerts":      "partial_mutate",
		"code_scanning_default_setup": "partial_mutate",
		"codeql_setup":                "partial_mutate",
	} {
		if found[capability] != mode {
			t.Fatalf("expected boundary %s=%s, got %q", capability, mode, found[capability])
		}
	}
}

func TestRelevantCapabilityBoundariesFiltersRequestedIDs(t *testing.T) {
	items := RelevantCapabilityBoundaries(PlatformGitHub, []string{"release", "pages", "advanced_security", "secret_scanning_alerts", "unknown"})
	if len(items) != 4 {
		t.Fatalf("expected 4 relevant boundaries, got %d", len(items))
	}
}

func TestCapabilityBoundariesIncludeGitLabAndBitbucketParityEntries(t *testing.T) {
	for _, tc := range []struct {
		platform   Platform
		capability string
		mode       string
	}{
		{platform: PlatformGitLab, capability: "merge_requests", mode: "partial_mutate"},
		{platform: PlatformGitLab, capability: "security", mode: "inspect_only"},
		{platform: PlatformBitbucket, capability: "pull_requests", mode: "partial_mutate"},
		{platform: PlatformBitbucket, capability: "repository_variables", mode: "partial_mutate"},
	} {
		boundary, ok := CapabilityBoundaryFor(tc.platform, tc.capability)
		if !ok {
			t.Fatalf("expected boundary for %s on %s", tc.capability, tc.platform)
		}
		if boundary.Mode != tc.mode {
			t.Fatalf("expected %s=%s on %s, got %s", tc.capability, tc.mode, tc.platform, boundary.Mode)
		}
	}
}
