package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet_LoadsData(t *testing.T) {
	c := Get()
	assert.NotEmpty(t, c.Placeholders.ExactPatterns, "exact patterns should be loaded")
	assert.NotEmpty(t, c.Placeholders.PathPatterns, "path patterns should be loaded")
	assert.NotEmpty(t, c.Subcommands, "subcommands should be loaded")
	assert.NotEmpty(t, c.WorkflowList(), "workflows should be loaded")
}

func TestIsPlaceholder_AngleBrackets(t *testing.T) {
	c := Get()
	assert.True(t, c.IsPlaceholder("<remote-url>"))
	assert.True(t, c.IsPlaceholder("<branch-name>"))
}

func TestIsPlaceholder_ExactPatterns(t *testing.T) {
	c := Get()
	assert.True(t, c.IsPlaceholder("git@github.com:your-username/your-repo.git"))
	assert.True(t, c.IsPlaceholder("https://example.com/repo"))
	assert.False(t, c.IsPlaceholder("git@github.com:realuser/realrepo.git"))
}

func TestIsPlaceholder_PathPatterns(t *testing.T) {
	c := Get()
	assert.True(t, c.IsPlaceholder("git@github.com:user/repo.git"))
	assert.True(t, c.IsPlaceholder("git@github.com:username/repo.git"))
}

func TestSubcommandLabel(t *testing.T) {
	c := Get()
	assert.Equal(t, "Remote URL", c.SubcommandLabel("remote"))
	assert.Equal(t, "Branch name", c.SubcommandLabel("checkout"))
	assert.Equal(t, "Tag name", c.SubcommandLabel("tag"))
	assert.Equal(t, "Repository URL", c.SubcommandLabel("clone"))
	assert.Equal(t, "", c.SubcommandLabel("nonexistent"))
}

func TestCommitInfo(t *testing.T) {
	c := Get()
	info := c.CommitInfo()
	assert.True(t, info.RequiresMessage)
	assert.Contains(t, info.MessageFlags, "-m")
	assert.Contains(t, info.SkipMessageFlags, "--amend")
}

func TestGuessLabel(t *testing.T) {
	c := Get()
	assert.Equal(t, "Remote URL", c.GuessLabel("https://github.com/foo/bar", "push"))
	assert.Equal(t, "Branch name", c.GuessLabel("feature/auth", "checkout"))
	assert.Equal(t, "Branch name", c.GuessLabel("<name>", "branch"))
	assert.Equal(t, "Tag name", c.GuessLabel("<version>", "tag"))
}

func TestDefaultPlaceholder(t *testing.T) {
	c := Get()
	assert.Equal(t, "git@github.com:user/repo.git", c.DefaultPlaceholder("remote", "URL", true))
	assert.Equal(t, "https://github.com/user/repo.git", c.DefaultPlaceholder("remote", "URL", false))
	assert.Equal(t, "Enter value...", c.DefaultPlaceholder("foo", "bar", false))
}
