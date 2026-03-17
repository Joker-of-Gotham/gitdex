package dotgitdex

import (
	"os"
	"path/filepath"
	"strings"
)

const dirName = ".gitdex"

// Manager manages the .gitdex/ directory and all files within it.
type Manager struct {
	Root     string
	RepoRoot string
}

// New creates a Manager rooted at repoRoot/.gitdex.
func New(repoRoot string) *Manager {
	return &Manager{Root: filepath.Join(repoRoot, dirName), RepoRoot: repoRoot}
}

// Init creates the .gitdex directory tree and ensures .gitdex/ is in .gitignore.
func (m *Manager) Init() error {
	for _, sub := range []string{
		m.MaintainDir(),
		m.KnowledgeDir(),
		m.GoalListDir(),
		m.ProposalDir(),
	} {
		if err := os.MkdirAll(sub, 0o755); err != nil {
			return err
		}
	}
	m.ensureGitignore()
	return nil
}

// ensureGitignore adds ".gitdex/" to .gitignore if not already present.
func (m *Manager) ensureGitignore() {
	gitignorePath := filepath.Join(m.RepoRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".gitdex/" || trimmed == ".gitdex" || trimmed == "/.gitdex/" || trimmed == "/.gitdex" {
			return
		}
	}
	entry := "\n# Gitdex state directory\n.gitdex/\n"
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		entry = "\n" + entry
	}
	_ = os.WriteFile(gitignorePath, []byte(content+entry), 0o644)
}

func (m *Manager) MaintainDir() string  { return filepath.Join(m.Root, "maintain") }
func (m *Manager) KnowledgeDir() string { return filepath.Join(m.Root, "maintain", "knowledge") }
func (m *Manager) GoalListDir() string  { return filepath.Join(m.Root, "goal-list") }
func (m *Manager) ProposalDir() string  { return filepath.Join(m.Root, "proposal") }

func (m *Manager) GitContentPath() string        { return filepath.Join(m.MaintainDir(), "git-content.txt") }
func (m *Manager) OutputPath() string             { return filepath.Join(m.MaintainDir(), "output.txt") }
func (m *Manager) IndexPath() string              { return filepath.Join(m.MaintainDir(), "index.yaml") }
func (m *Manager) GoalListPath() string           { return filepath.Join(m.GoalListDir(), "goal-list.md") }
func (m *Manager) CreativeProposalPath() string   { return filepath.Join(m.ProposalDir(), "creative-proposal.md") }
func (m *Manager) DiscardedProposalPath() string  { return filepath.Join(m.ProposalDir(), "discarded-proposal.md") }
