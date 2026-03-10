package components

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/stretchr/testify/assert"
)

func TestAreasTree_ViewIncludesCoreSections(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{
			Name:     "master",
			Upstream: "origin/master",
			Ahead:    1,
			Behind:   2,
		},
		CommitCount: 5,
		WorkingTree: []git.FileStatus{
			{Path: "main.go", WorktreeCode: git.StatusModified},
			{Path: "README.md", WorktreeCode: git.StatusUntracked},
		},
		StagingArea: []git.FileStatus{
			{Path: "go.mod", StagingCode: git.StatusAdded},
		},
		RemoteInfos: []git.RemoteInfo{
			{
				Name:                "origin",
				FetchURL:            "git@github.com:user/repo.git",
				PushURL:             "git@github.com:user/repo.git",
				FetchURLValid:       true,
				PushURLValid:        true,
				ReachabilityChecked: true,
				Reachable:           true,
			},
		},
	}

	out := NewAreasTree(state).SetWidth(42).View()
	assert.Contains(t, out, "Working Directory")
	assert.Contains(t, out, "Staging Area")
	assert.Contains(t, out, "Local Repository")
	assert.Contains(t, out, "origin (remote)")
	assert.Contains(t, out, "SSH")
}

func TestAreasTree_NoRemoteConfigured(t *testing.T) {
	state := &status.GitState{
		LocalBranch: git.BranchInfo{Name: "master"},
		CommitCount: 1,
	}

	out := NewAreasTree(state).SetWidth(38).View()
	assert.Contains(t, out, "Remote")
	assert.Contains(t, out, "not configured")
}
