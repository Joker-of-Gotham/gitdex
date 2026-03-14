package context

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func TestRetrieveWithGoalMatchesGoalContainsScenario(t *testing.T) {
	r := NewRetriever()
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}

	fragments := r.RetrieveWithGoal(state, "我想配置 GitHub Pages 自定义域名和 build history")
	if len(fragments) == 0 {
		t.Fatal("expected knowledge fragments")
	}

	found := false
	for _, fragment := range fragments {
		if fragment.ScenarioID == "platform_github#github_pages_surface" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected github_pages_surface fragment")
	}
}

func TestRetrieveWithGoalMatchesPlatformListTrigger(t *testing.T) {
	r := NewRetriever()
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@bitbucket.org:team/repo.git",
		}},
	}

	fragments := r.RetrieveWithGoal(state, "audit webhook permissions")
	if len(fragments) == 0 {
		t.Fatal("expected knowledge fragments")
	}

	found := false
	for _, fragment := range fragments {
		if fragment.ScenarioID == "platform_bitbucket#bitbucket_admin_surface" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected bitbucket_admin_surface fragment")
	}
}
