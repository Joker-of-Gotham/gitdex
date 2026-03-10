package status

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestDetectAnomalies_NilInput(t *testing.T) {
	assert.Empty(t, DetectAnomalies(nil, nil))
	assert.Empty(t, DetectAnomalies(&GitState{}, nil))
	assert.Empty(t, DetectAnomalies(nil, &GitState{}))
}

func TestDetectAnomalies_BranchChange(t *testing.T) {
	prev := &GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
	}
	curr := &GitState{
		LocalBranch: git.BranchInfo{Name: "feature"},
	}
	a := DetectAnomalies(prev, curr)
	assert.Len(t, a, 1)
	assert.Contains(t, a[0], "main")
	assert.Contains(t, a[0], "feature")
}

func TestDetectAnomalies_DetachedHEAD(t *testing.T) {
	prev := &GitState{
		LocalBranch: git.BranchInfo{Name: "main", IsDetached: false},
	}
	curr := &GitState{
		LocalBranch: git.BranchInfo{IsDetached: true},
		HeadRef:     "abc123",
	}
	a := DetectAnomalies(prev, curr)
	assert.NotEmpty(t, a)
}

func TestDetectAnomalies_StashDecreased(t *testing.T) {
	prev := &GitState{
		StashStack: []git.StashEntry{{Index: 0}, {Index: 1}},
	}
	curr := &GitState{
		StashStack: []git.StashEntry{{Index: 0}},
	}
	a := DetectAnomalies(prev, curr)
	assert.Len(t, a, 1)
	assert.Contains(t, a[0], "stash")
}

func TestDetectAnomalies_NoAnomalies(t *testing.T) {
	prev := &GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		StashStack:  []git.StashEntry{{Index: 0}},
	}
	curr := &GitState{
		LocalBranch: git.BranchInfo{Name: "main"},
		StashStack:  []git.StashEntry{{Index: 0}, {Index: 1}},
	}
	a := DetectAnomalies(prev, curr)
	assert.Empty(t, a)
}
