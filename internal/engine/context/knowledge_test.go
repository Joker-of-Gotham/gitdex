package context

import "testing"

func TestLoadKnowledgeBaseIncludesGitHubAndBitbucketPlatformScenarios(t *testing.T) {
	kb := LoadKnowledgeBase()
	if kb == nil || len(kb.Scenarios) == 0 {
		t.Fatal("expected knowledge scenarios")
	}

	foundGitHub := false
	foundBitbucket := false
	for _, scenario := range kb.Scenarios {
		switch scenario.ID {
		case "github_release_surface":
			foundGitHub = true
		case "bitbucket_pr_pipeline_flow":
			foundBitbucket = true
		}
	}
	if !foundGitHub {
		t.Fatal("expected github_release_surface scenario")
	}
	if !foundBitbucket {
		t.Fatal("expected bitbucket_pr_pipeline_flow scenario")
	}
}
