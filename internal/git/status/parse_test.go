package status

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatusV2_Empty(t *testing.T) {
	out := ""
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Empty(t, state.WorkingTree)
	assert.Empty(t, state.StagingArea)
}

func TestParseStatusV2_BranchHeaders(t *testing.T) {
	out := `# branch.oid abc123def456
# branch.head main
# branch.upstream origin/main
# branch.ab +3 -1
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "abc123def456", state.HeadRef)
	assert.Equal(t, "main", state.LocalBranch.Name)
	assert.Equal(t, "origin/main", state.LocalBranch.Upstream)
	assert.Equal(t, 3, state.LocalBranch.Ahead)
	assert.Equal(t, 1, state.LocalBranch.Behind)
	assert.False(t, state.LocalBranch.IsDetached)
	assert.NotNil(t, state.UpstreamState)
	assert.Equal(t, "origin/main", state.UpstreamState.Name)
	assert.Equal(t, 3, state.UpstreamState.Ahead)
	assert.Equal(t, 1, state.UpstreamState.Behind)
}

func TestParseStatusV2_DetachedHead(t *testing.T) {
	out := `# branch.oid abc123
# branch.head (detached)
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	assert.True(t, state.LocalBranch.IsDetached)
	assert.Empty(t, state.LocalBranch.Name)
}

func TestParseStatusV2_InitialCommit(t *testing.T) {
	out := `# branch.oid (initial)
# branch.head main
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	assert.Equal(t, "main", state.LocalBranch.Name)
}

func TestParseStatusV2_OrdinaryModified(t *testing.T) {
	// .M = unstaged modification
	out := `# branch.oid abc123
# branch.head main
1 .M N... 100644 100644 100644 111111 111111 foo.go
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.WorkingTree, 1)
	assert.Equal(t, "foo.go", state.WorkingTree[0].Path)
	assert.Equal(t, git.StatusUnmodified, state.WorkingTree[0].StagingCode)
	assert.Equal(t, git.StatusModified, state.WorkingTree[0].WorktreeCode)
	assert.Empty(t, state.StagingArea)
}

func TestParseStatusV2_OrdinaryStaged(t *testing.T) {
	// M. = staged modification
	out := `# branch.oid abc123
# branch.head main
1 M. N... 100644 100644 100644 111111 222222 bar.go
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.StagingArea, 1)
	assert.Equal(t, "bar.go", state.StagingArea[0].Path)
	assert.Equal(t, git.StatusModified, state.StagingArea[0].StagingCode)
	assert.Equal(t, git.StatusUnmodified, state.StagingArea[0].WorktreeCode)
	assert.Empty(t, state.WorkingTree)
}

func TestParseStatusV2_OrdinaryBothStagedAndModified(t *testing.T) {
	// MM = staged and worktree modified
	out := `# branch.oid abc123
# branch.head main
1 MM N... 100644 100644 100644 111111 222222 baz.go
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.StagingArea, 1)
	require.Len(t, state.WorkingTree, 1)
	assert.Equal(t, "baz.go", state.StagingArea[0].Path)
	assert.Equal(t, "baz.go", state.WorkingTree[0].Path)
	assert.Equal(t, git.StatusModified, state.StagingArea[0].StagingCode)
	assert.Equal(t, git.StatusModified, state.WorkingTree[0].WorktreeCode)
}

func TestParseStatusV2_OrdinaryDeleted(t *testing.T) {
	// .D = deleted in worktree
	out := `1 .D N... 100644 100644 100644 111111 111111 deleted.go
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.WorkingTree, 1)
	assert.Equal(t, "deleted.go", state.WorkingTree[0].Path)
	assert.Equal(t, git.StatusDeleted, state.WorkingTree[0].WorktreeCode)
}

func TestParseStatusV2_OrdinaryAdded(t *testing.T) {
	// A. = added to index
	out := `1 A. N... 000000 100644 100644 000000 222222 newfile.go
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.StagingArea, 1)
	assert.Equal(t, "newfile.go", state.StagingArea[0].Path)
	assert.Equal(t, git.StatusAdded, state.StagingArea[0].StagingCode)
}

func TestParseStatusV2_Untracked(t *testing.T) {
	out := `# branch.oid abc123
# branch.head main
? untracked.go
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.WorkingTree, 1)
	assert.Equal(t, "untracked.go", state.WorkingTree[0].Path)
	assert.Equal(t, git.StatusUntracked, state.WorkingTree[0].StagingCode)
	assert.Equal(t, git.StatusUntracked, state.WorkingTree[0].WorktreeCode)
	assert.Empty(t, state.StagingArea)
}

func TestParseStatusV2_Ignored(t *testing.T) {
	out := `! ignored.log
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.WorkingTree, 1)
	assert.Equal(t, "ignored.log", state.WorkingTree[0].Path)
	assert.Equal(t, git.StatusIgnored, state.WorkingTree[0].WorktreeCode)
}

func TestParseStatusV2_Renamed(t *testing.T) {
	// porcelain v2 spec: 2 <XY> <sub> ... <X><score> <path><TAB><origPath>
	// path = destination (new name), origPath = source (old name)
	out := `2 R. N... 100644 100644 100644 111111 222222 R100 newname.go	oldname.go
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.StagingArea, 1)
	assert.Equal(t, "newname.go", state.StagingArea[0].Path)
	assert.Equal(t, "oldname.go", state.StagingArea[0].OrigPath)
	assert.Equal(t, git.StatusRenamed, state.StagingArea[0].StagingCode)
}

func TestParseStatusV2_Unmerged(t *testing.T) {
	out := `u UU N... 100644 100644 100644 100644 111111 222222 333333 conflicted.go
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.StagingArea, 1)
	require.Len(t, state.WorkingTree, 1)
	assert.Equal(t, "conflicted.go", state.StagingArea[0].Path)
	assert.Equal(t, git.StatusUnmerged, state.StagingArea[0].StagingCode)
}

func TestParseStatusV2_FullSample(t *testing.T) {
	out := `# branch.oid abc123def
# branch.head feature
# branch.upstream origin/feature
# branch.ab +2 -0
1 .M N... 100644 100644 100644 111111 111111 modified.go
1 M. N... 100644 100644 100644 111111 222222 staged.go
? newfile.txt
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)

	// Branch info
	assert.Equal(t, "feature", state.LocalBranch.Name)
	assert.Equal(t, "origin/feature", state.LocalBranch.Upstream)
	assert.Equal(t, 2, state.LocalBranch.Ahead)
	assert.Equal(t, 0, state.LocalBranch.Behind)

	// Working tree: modified.go (unstaged .M), newfile.txt (untracked). staged.go (M.) is only in staging.
	assert.Len(t, state.WorkingTree, 2)

	// Staging area: staged.go
	assert.Len(t, state.StagingArea, 1)
	assert.Equal(t, "staged.go", state.StagingArea[0].Path)
}

func TestParseStatusV2_QuotedPath(t *testing.T) {
	out := `? "file with spaces.go"
`
	state, err := ParseStatusV2(out)
	require.NoError(t, err)
	require.Len(t, state.WorkingTree, 1)
	assert.Equal(t, "file with spaces.go", state.WorkingTree[0].Path)
}
