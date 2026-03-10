package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
)

// CollectFileContext reads important files and returns their contents.
func CollectFileContext(ctx context.Context, state *status.GitState) *prompt.FileContext {
	fc := &prompt.FileContext{
		Files: make(map[string]string),
	}

	// Always try to read .gitignore
	if state.HasGitIgnore {
		if content, err := readFile(".gitignore"); err == nil {
			fc.Files[".gitignore"] = content
		}
	}

	// Read other important files if they exist
	importantPaths := []string{
		".gitattributes",
		"README.md",
		"README",
		"package.json",
		"go.mod",
		"pyproject.toml",
		"requirements.txt",
		"Cargo.toml",
		"Makefile",
	}

	for _, path := range importantPaths {
		if content, err := readFile(path); err == nil {
			fc.Files[path] = content
		}
	}

	return fc
}

func readFile(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	// Limit file size to prevent context explosion
	content := string(data)
	if len(content) > 10000 {
		content = content[:10000] + "\n... (truncated)"
	}
	return content, nil
}
