package platform

import "testing"

func TestCapabilityCatalogGitHubIncludesAdvancedSecurity(t *testing.T) {
	items := CapabilityCatalog(PlatformGitHub)
	if len(items) == 0 {
		t.Fatal("expected github capabilities")
	}
	found := false
	for _, item := range items {
		if item.ID == "advanced_security" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected advanced_security capability")
	}
	found = false
	for _, item := range items {
		if item.ID == "pull_request" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected pull_request capability")
	}
	for _, capabilityID := range []string{"dependabot_posture", "secret_scanning_settings", "secret_scanning_alerts", "code_scanning_tool_settings", "code_scanning_default_setup", "codeql_setup"} {
		found = false
		for _, item := range items {
			if item.ID == capabilityID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %s capability", capabilityID)
		}
	}
}

func TestRecommendCapabilityPlaybooksMatchesChineseGoal(t *testing.T) {
	playbooks := RecommendCapabilityPlaybooks(PlatformGitHub, "我需要配置 deployment、Pages、Actions secrets 和 webhooks", 5)
	if len(playbooks) == 0 {
		t.Fatal("expected recommended playbooks")
	}
	foundDeployment := false
	foundPages := false
	foundSecrets := false
	for _, playbook := range playbooks {
		switch playbook.ID {
		case "deployment":
			foundDeployment = true
		case "pages":
			foundPages = true
		case "actions_secrets_variables":
			foundSecrets = true
		}
	}
	if !foundDeployment {
		t.Fatal("expected deployment playbook")
	}
	if !foundPages {
		t.Fatal("expected pages playbook")
	}
	if !foundSecrets {
		t.Fatal("expected actions_secrets_variables playbook")
	}
}

func TestCapabilityProbesIncludeThirdBatchSecuritySurfaces(t *testing.T) {
	probes := CapabilityProbes(PlatformGitHub)
	found := map[string]bool{}
	for _, probe := range probes {
		found[probe.CapabilityID] = true
	}
	for _, capability := range []string{"actions", "codespaces", "security", "check_runs_failure_threshold", "advanced_security", "dependabot_posture", "dependency_graph", "dependabot_security_updates", "secret_scanning_settings", "secret_scanning_alerts", "code_scanning_tool_settings", "code_scanning_default_setup", "codeql_setup", "codeql_analysis", "push_protection", "dependabot_config", "copilot_seat_management", "pull_request"} {
		if !found[capability] {
			t.Fatalf("expected probe for %s", capability)
		}
	}
}
