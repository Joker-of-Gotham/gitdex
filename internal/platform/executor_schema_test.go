package platform

import "testing"

func TestRecommendedExecutorSchemasFavorsPlaybooksAndGoal(t *testing.T) {
	hints := RecommendedExecutorSchemas(
		PlatformGitHub,
		"请帮我发布 release 并检查 release asset 和 release notes",
		[]string{"release", "pages"},
		4,
	)
	if len(hints) == 0 {
		t.Fatal("expected executor schema hints")
	}
	if hints[0].CapabilityID != "release" {
		t.Fatalf("expected release to be first, got %s", hints[0].CapabilityID)
	}
	if hints[0].Example == "" {
		t.Fatal("expected release example guidance")
	}
}

func TestExecutorSchemaForPRReviewIncludesPullNumber(t *testing.T) {
	hint, ok := ExecutorSchemaFor(PlatformGitHub, "pr_review")
	if !ok {
		t.Fatal("expected pr_review schema")
	}
	found := false
	for _, key := range hint.ScopeKeys {
		if key == "pull_number" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected pull_number scope key")
	}
}

func TestExecutorSchemaForPullRequestIncludesLifecycleOps(t *testing.T) {
	hint, ok := ExecutorSchemaFor(PlatformGitHub, "pull_request")
	if !ok {
		t.Fatal("expected pull_request schema")
	}
	required := map[string]bool{
		"create":            false,
		"update":            false,
		"close":             false,
		"reopen":            false,
		"merge":             false,
		"enable_auto_merge": false,
	}
	for _, op := range hint.MutateOps {
		if _, ok := required[op]; ok {
			required[op] = true
		}
	}
	for op, found := range required {
		if !found {
			t.Fatalf("expected mutate op %s", op)
		}
	}
}

func TestExecutorSchemaForAdvancedSecurityFamilyExists(t *testing.T) {
	for _, capability := range []string{"actions", "codespaces", "security", "check_runs_failure_threshold", "advanced_security", "dependabot_posture", "secret_scanning_settings", "secret_scanning_alerts", "dependabot_alerts", "code_scanning", "code_scanning_tool_settings", "code_scanning_default_setup", "codeql_setup", "secret_protection", "copilot_code_review", "dependabot_config", "copilot_seat_management", "pull_request"} {
		if _, ok := ExecutorSchemaFor(PlatformGitHub, capability); !ok {
			t.Fatalf("expected schema for %s", capability)
		}
	}
}

func TestExecutorSchemaForGitLabAndBitbucketParitySurfaces(t *testing.T) {
	for _, tc := range []struct {
		platform   Platform
		capability string
		mutateOp   string
	}{
		{platform: PlatformGitLab, capability: "merge_requests", mutateOp: "create"},
		{platform: PlatformGitLab, capability: "pipelines", mutateOp: "retry"},
		{platform: PlatformBitbucket, capability: "pull_requests", mutateOp: "create"},
		{platform: PlatformBitbucket, capability: "repository_variables", mutateOp: "update"},
	} {
		hint, ok := ExecutorSchemaFor(tc.platform, tc.capability)
		if !ok {
			t.Fatalf("expected schema for %s on %s", tc.capability, tc.platform)
		}
		found := false
		for _, op := range hint.MutateOps {
			if op == tc.mutateOp {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected mutate op %s for %s on %s", tc.mutateOp, tc.capability, tc.platform)
		}
	}
}
