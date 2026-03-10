package analyzer

import (
	"os"
	"path/filepath"
)

// DetectProjectType checks for marker files (go.mod, package.json, requirements.txt, Cargo.toml).
// Returns "go", "node", "python", "rust", or "" if unknown.
func DetectProjectType(dir string) string {
	if dir == "" {
		// Use current working directory
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	markers := map[string]string{
		"go.mod":           "go",
		"package.json":     "node",
		"requirements.txt": "python",
		"Cargo.toml":       "rust",
	}
	for name, pt := range markers {
		path := filepath.Join(dir, name)
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			return pt
		}
	}
	return ""
}

// SuggestGitignore returns .gitignore content for the given project type.
func SuggestGitignore(projectType string) string {
	switch projectType {
	case "go":
		return `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output
/bin/
/dist/

# Dependency directories
/vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo
`
	case "node":
		return `# Dependencies
node_modules/

# Build output
dist/
build/
.next/
out/

# Logs
*.log
npm-debug.log*

# Environment
.env
.env.local
`
	case "python":
		return `# Byte-compiled
__pycache__/
*.py[cod]
*$py.class

# Virtual env
venv/
.venv/
env/

# Distribution
dist/
build/
*.egg-info/

# IDE
.idea/
.vscode/
`
	case "rust":
		return `# Build
target/
Cargo.lock

# IDE
.idea/
.vscode/
`
	default:
		return `# OS
.DS_Store
Thumbs.db

# IDE
.idea/
.vscode/
*.swp
*.swo
`
	}
}
