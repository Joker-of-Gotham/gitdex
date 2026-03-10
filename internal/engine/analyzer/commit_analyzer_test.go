package analyzer

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommitAnalyzer(t *testing.T) {
	a := NewCommitAnalyzer()
	require.NotNil(t, a)
}

func TestGenerateMessage_EmptyStaged(t *testing.T) {
	a := NewCommitAnalyzer()
	assert.Empty(t, a.GenerateMessage(nil))
	assert.Empty(t, a.GenerateMessage([]git.FileStatus{}))
}

func TestGenerateMessage_SingleNewFile(t *testing.T) {
	a := NewCommitAnalyzer()
	staged := []git.FileStatus{
		{Path: "cmd/main.go", StagingCode: git.StatusAdded},
	}
	msg := a.GenerateMessage(staged)
	assert.Equal(t, "feat(cmd): add main.go", msg)
}

func TestGenerateMessage_SingleModifiedFile(t *testing.T) {
	a := NewCommitAnalyzer()
	staged := []git.FileStatus{
		{Path: "internal/parser.go", StagingCode: git.StatusModified},
	}
	msg := a.GenerateMessage(staged)
	assert.Equal(t, "fix(internal): update parser.go", msg)
}

func TestGenerateMessage_SingleDeletedFile(t *testing.T) {
	a := NewCommitAnalyzer()
	staged := []git.FileStatus{
		{Path: "old/deprecated.go", StagingCode: git.StatusDeleted},
	}
	msg := a.GenerateMessage(staged)
	assert.Equal(t, "chore(old): remove deprecated.go", msg)
}

func TestGenerateMessage_SingleFileNoScope(t *testing.T) {
	a := NewCommitAnalyzer()
	staged := []git.FileStatus{
		{Path: "README.md", StagingCode: git.StatusAdded},
	}
	msg := a.GenerateMessage(staged)
	assert.Equal(t, "feat: add README.md", msg)
}

func TestGenerateMessage_MultipleFiles(t *testing.T) {
	a := NewCommitAnalyzer()
	staged := []git.FileStatus{
		{Path: "a.go", StagingCode: git.StatusModified},
		{Path: "b.go", StagingCode: git.StatusModified},
	}
	msg := a.GenerateMessage(staged)
	assert.Equal(t, "fix: update 2 files", msg)
}

func TestGenerateMessage_MixedNewAndModified(t *testing.T) {
	a := NewCommitAnalyzer()
	staged := []git.FileStatus{
		{Path: "new.go", StagingCode: git.StatusAdded},
		{Path: "old.go", StagingCode: git.StatusModified},
	}
	msg := a.GenerateMessage(staged)
	// hasMod is true, so returns "fix"
	assert.Equal(t, "fix: update 2 files", msg)
}
